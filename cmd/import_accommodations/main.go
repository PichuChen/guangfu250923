package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"guangfu250923/internal/config"
	"guangfu250923/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Column mapping based on the CSV header observed.
// Header (trimmed): 鄉鎮,民宿名稱,尚有空房？,開放期間,限制,聯繫方式,時間、房型,地址,經緯度,費用,資訊來源,其他
// We map to accommodations columns:
// township -> 鄉鎮
// name -> 民宿名稱
// has_vacancy -> 尚有空房？ (normalize: 有空房 -> available, 已滿 -> full, 未知/空白 -> unknown)
// available_period -> 開放期間
// restrictions -> 限制 (nullable)
// contact_info -> 聯繫方式 (sanitize newlines)
// room_info -> 時間、房型
// address -> 地址
// pricing -> 費用 (blank -> "")
// info_source -> 資訊來源 (nullable)
// notes -> 其他 (nullable)
// lat,lng extracted from 經緯度 if format like: [lat, lng]
// status -> 固定 "active"
// capacity, registration_method, facilities, distance_to_disaster_area left NULL / empty

// coordPatterns tries to be tolerant of various coordinate string formats:
// Examples supported now:
//
//	[23.66, 121.42]
//	23.66,121.42
//	23.66 / 121.42
//	23.66 121.42
//	lat:23.66 lng:121.42
//
// We simply extract the first two float-looking tokens and treat them as (lat,lng).
var floatTokenRe = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)

func normalizeVacancy(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\uFEFF") // remove BOM if any
	switch s {
	case "有空房", "有", "空房":
		return "available"
	case "已滿", "滿", "額滿":
		return "full"
	case "未知", "?":
		return "unknown"
	case "", " ":
		return "unknown"
	default:
		return s // keep raw (e.g., 特殊標註)
	}
}

func parseCoords(raw string) (lat *float64, lng *float64) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, nil
	}
	// Replace common separators with space for simpler tokenization
	replacers := []string{",", ";", "/", "|"}
	for _, r := range replacers {
		s = strings.ReplaceAll(s, r, " ")
	}
	// Extract float tokens
	tokens := floatTokenRe.FindAllString(s, -1)
	if len(tokens) < 2 {
		return nil, nil
	}
	var la, ln float64
	fmt.Sscanf(tokens[0], "%f", &la)
	fmt.Sscanf(tokens[1], "%f", &ln)
	// If first number looks like longitude (>90 abs) and second looks like latitude, swap
	if (la < -90 || la > 90) && (ln >= -90 && ln <= 90) {
		la, ln = ln, la
	}
	return &la, &ln
}

func main() {
	csvPath := flag.String("csv", "scripts/花蓮光復鄉-救災志工_物資募集資訊總表 - 【災民志工相關】優待住宿(災民_志工).csv", "Path to accommodations CSV")
	dryRun := flag.Bool("dry", false, "Dry run (do not insert)")
	flag.Parse()

	cfg := config.Load()
	pool, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	f, err := os.Open(*csvPath)
	if err != nil {
		log.Fatalf("open csv: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(bufio.NewReader(f))
	reader.FieldsPerRecord = -1 // allow variable
	line := 0
	inserted := 0
	for {
		rec, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatalf("csv read error line %d: %v", line+1, err)
		}
		line++
		// skip header if first field contains 鄉鎮
		if line == 1 && strings.Contains(rec[0], "鄉鎮") {
			continue
		}
		// Ensure at least 12 columns (pad if shorter)
		if len(rec) < 12 {
			pad := make([]string, 12-len(rec))
			rec = append(rec, pad...)
		}
		for i := range rec {
			rec[i] = strings.TrimSpace(rec[i])
		}
		township := rec[0]
		name := rec[1]
		if township == "" && name == "" {
			continue // empty line
		}
		hasVacancy := normalizeVacancy(rec[2])
		availablePeriod := rec[3]
		restrictions := nullable(rec[4])
		contact := singleLine(rec[5])
		roomInfo := nullable(rec[6])
		address := rec[7]
		coordsRaw := rec[8]
		pricing := rec[9]
		infoSource := nullable(rec[10])
		notes := nullable(rec[11])
		lat, lng := parseCoords(coordsRaw)
		status := "active"

		if *dryRun {
			fmt.Printf("[DRY] %s %s vacancy=%s period=%s addr=%s lat,lng=%v,%v\n", township, name, hasVacancy, availablePeriod, address, lat, lng)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := upsertAccommodation(ctx, pool, township, name, hasVacancy, availablePeriod, restrictions, contact, roomInfo, address, pricing, infoSource, notes, status, lat, lng); err != nil {
			cancel()
			log.Fatalf("insert line %d (%s): %v", line, name, err)
		}
		cancel()
		inserted++
	}
	log.Printf("done. inserted %d rows", inserted)
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func singleLine(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "; ")
	return strings.TrimSpace(s)
}

func upsertAccommodation(ctx context.Context, pool *pgxpool.Pool, township, name, hasVacancy, availablePeriod string, restrictions *string, contact string, roomInfo *string, address, pricing string, infoSource, notes *string, status string, lat, lng *float64) error {
	// Try to find existing by (township,name,address) heuristic
	var existingID string
	err := pool.QueryRow(ctx, `select id from accommodations where township=$1 and name=$2 and address=$3 limit 1`, township, name, address).Scan(&existingID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if existingID != "" {
		// update minimal fields and coordinates JSONB
		var coords *string
		if lat != nil || lng != nil {
			coord := struct {
				Lat *float64 `json:"lat"`
				Lng *float64 `json:"lng"`
			}{Lat: lat, Lng: lng}
			if b, err := json.Marshal(coord); err == nil {
				s := string(b)
				coords = &s
			}
		}
		_, err = pool.Exec(ctx, `update accommodations set has_vacancy=$1,available_period=$2,contact_info=$3,room_info=$4,pricing=$5,info_source=$6,notes=$7,coordinates=$8::jsonb,updated_at=now() where id=$9`, hasVacancy, availablePeriod, contact, roomInfo, pricing, infoSource, notes, coords, existingID)
		return err
	}
	// insert
	// insert with coordinates JSONB
	var coords *string
	if lat != nil || lng != nil {
		coord := struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		}{Lat: lat, Lng: lng}
		if b, err := json.Marshal(coord); err == nil {
			s := string(b)
			coords = &s
		}
	}
	_, err = pool.Exec(ctx, `insert into accommodations(township,name,has_vacancy,available_period,restrictions,contact_info,room_info,address,pricing,info_source,notes,status,coordinates) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb)`,
		township, name, hasVacancy, availablePeriod, restrictions, contact, roomInfo, address, pricing, infoSource, notes, status, coords)
	return err
}
