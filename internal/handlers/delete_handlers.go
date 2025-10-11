package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

func deleteByID(c *gin.Context, h *Handler, table string) {
	id := c.Param("id")
	tag, err := h.pool.Exec(context.Background(), "delete from "+table+" where id=$1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteShelter(c *gin.Context)        { deleteByID(c, h, "shelters") }
func (h *Handler) DeleteMedicalStation(c *gin.Context) { deleteByID(c, h, "medical_stations") }
func (h *Handler) DeleteMentalHealthResource(c *gin.Context) {
	deleteByID(c, h, "mental_health_resources")
}
func (h *Handler) DeleteAccommodation(c *gin.Context)      { deleteByID(c, h, "accommodations") }
func (h *Handler) DeleteShowerStation(c *gin.Context)      { deleteByID(c, h, "shower_stations") }
func (h *Handler) DeleteWaterRefillStation(c *gin.Context) { deleteByID(c, h, "water_refill_stations") }
func (h *Handler) DeleteRestroom(c *gin.Context)           { deleteByID(c, h, "restrooms") }
func (h *Handler) DeleteVolunteerOrg(c *gin.Context)       { deleteByID(c, h, "volunteer_organizations") }
func (h *Handler) DeleteHumanResource(c *gin.Context)      { deleteByID(c, h, "human_resources") }
func (h *Handler) DeleteSupply(c *gin.Context)             { deleteByID(c, h, "supplies") }
func (h *Handler) DeleteSupplyItem(c *gin.Context)         { deleteByID(c, h, "supply_items") }
func (h *Handler) DeleteReport(c *gin.Context)             { deleteByID(c, h, "reports") }
func (h *Handler) DeletePlace(c *gin.Context)              { deleteByID(c, h, "places") }
func (h *Handler) DeleteRequirementsHR(c *gin.Context)     { deleteByID(c, h, "requirements_hr") }
func (h *Handler) DeleteRequirementsSupplies(c *gin.Context) { deleteByID(c, h, "requirements_supplies") }
