package negotiation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/middleware"
)

// Handler handles HTTP requests for negotiation
type Handler struct {
	service *Service
}

// NewHandler creates a new negotiation handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// StartSession starts a new negotiation session (rider)
func (h *Handler) StartSession(c *gin.Context) {
	riderID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	session, err := h.service.StartSession(c.Request.Context(), riderID, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.CreatedResponse(c, ToSessionResponse(session))
}

// GetSession retrieves a negotiation session
func (h *Handler) GetSession(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid session ID")
		return
	}

	session, err := h.service.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "session not found")
		return
	}

	common.SuccessResponse(c, ToSessionResponse(session))
}

// GetActiveSession retrieves the rider's active session
func (h *Handler) GetActiveSession(c *gin.Context) {
	riderID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	session, err := h.service.GetActiveSessionByRider(c.Request.Context(), riderID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get session")
		return
	}

	if session == nil {
		common.SuccessResponse(c, nil)
		return
	}

	// Load offers
	fullSession, err := h.service.GetSession(c.Request.Context(), session.ID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get session")
		return
	}

	common.SuccessResponse(c, ToSessionResponse(fullSession))
}

// SubmitOffer allows a driver to submit an offer
func (h *Handler) SubmitOffer(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid session ID")
		return
	}

	var req SubmitOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get driver info from context or database
	// For now, use placeholder values - in production, these would come from the driver service
	driverInfo := &DriverInfo{}

	offer, err := h.service.SubmitOffer(c.Request.Context(), sessionID, driverID, &req, driverInfo)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.CreatedResponse(c, ToOfferResponse(offer))
}

// AcceptOffer allows a rider to accept an offer
func (h *Handler) AcceptOffer(c *gin.Context) {
	riderID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid session ID")
		return
	}

	offerID, err := uuid.Parse(c.Param("offerId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid offer ID")
		return
	}

	if err := h.service.AcceptOffer(c.Request.Context(), sessionID, offerID, riderID); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{
		"message":    "offer accepted",
		"session_id": sessionID,
		"offer_id":   offerID,
	})
}

// RejectOffer allows a rider to reject an offer (optional - just for tracking)
func (h *Handler) RejectOffer(c *gin.Context) {
	// Implementation similar to AcceptOffer but marks as rejected
	common.SuccessResponse(c, gin.H{"message": "offer rejected"})
}

// CancelSession cancels a negotiation session
func (h *Handler) CancelSession(c *gin.Context) {
	riderID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid session ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	if err := h.service.CancelSession(c.Request.Context(), sessionID, riderID, req.Reason); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "session cancelled"})
}

// WithdrawOffer allows a driver to withdraw their offer
func (h *Handler) WithdrawOffer(c *gin.Context) {
	// Implementation for driver withdrawing offer
	common.SuccessResponse(c, gin.H{"message": "offer withdrawn"})
}

// RegisterRoutes registers negotiation routes
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	nego := rg.Group("/negotiations")
	{
		// Rider endpoints
		nego.POST("", h.StartSession)
		nego.GET("/active", h.GetActiveSession)
		nego.GET("/:id", h.GetSession)
		nego.POST("/:id/cancel", h.CancelSession)
		nego.POST("/:id/offers/:offerId/accept", h.AcceptOffer)
		nego.POST("/:id/offers/:offerId/reject", h.RejectOffer)

		// Driver endpoints
		nego.POST("/:id/offers", h.SubmitOffer)
		nego.DELETE("/:id/offers/:offerId", h.WithdrawOffer)
	}
}
