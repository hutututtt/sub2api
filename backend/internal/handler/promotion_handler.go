package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const promotionVisitorCookie = "promotion_visitor"

type PromotionHandler struct{ service *service.PromotionService }

func NewPromotionHandler(s *service.PromotionService) *PromotionHandler {
	return &PromotionHandler{service: s}
}

type promotionVisitRequest struct {
	Code        string `json:"code" binding:"required"`
	EventID     string `json:"event_id"`
	LandingPath string `json:"landing_path"`
}

func (h *PromotionHandler) TrackVisit(c *gin.Context) {
	var req promotionVisitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid promotion visit")
		return
	}
	if isPromotionCrawler(c.GetHeader("User-Agent")) {
		c.Status(http.StatusNoContent)
		return
	}
	token, err := c.Cookie(promotionVisitorCookie)
	if err != nil || len(token) < 40 {
		raw := make([]byte, 32)
		if _, err = rand.Read(raw); err != nil {
			response.InternalError(c, "unable to start promotion session")
			return
		}
		token = base64.RawURLEncoding.EncodeToString(raw)
		secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(promotionVisitorCookie, token, 120*24*60*60, "/", "", secure, true)
	}
	eventID := req.EventID
	if _, err = uuid.Parse(eventID); err != nil {
		eventID = uuid.NewString()
	}
	tracked, err := h.service.Track(c.Request.Context(), promotionVisitorHash(token), service.PromotionVisit{
		Code: req.Code, EventID: eventID, LandingPath: req.LandingPath, ReferrerHost: c.GetHeader("Referer"), Device: promotionDevice(c.GetHeader("User-Agent")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"tracked": tracked})
}

func (h *PromotionHandler) Redirect(c *gin.Context) {
	code := c.Param("code")
	token, err := c.Cookie(promotionVisitorCookie)
	if err != nil || len(token) < 40 {
		raw := make([]byte, 32)
		if _, err = rand.Read(raw); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		token = base64.RawURLEncoding.EncodeToString(raw)
	}
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(promotionVisitorCookie, token, 120*24*60*60, "/", "", secure, true)
	if !isPromotionCrawler(c.GetHeader("User-Agent")) {
		_, err = h.service.Track(c.Request.Context(), promotionVisitorHash(token), service.PromotionVisit{Code: code, EventID: uuid.NewString(), LandingPath: "/", ReferrerHost: c.GetHeader("Referer"), Device: promotionDevice(c.GetHeader("User-Agent"))})
	}
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	channels, err := h.service.Channels(c.Request.Context())
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	for _, item := range channels {
		if item.Code == code && item.Enabled {
			c.Redirect(http.StatusFound, item.LandingPath)
			return
		}
	}
	c.AbortWithStatus(http.StatusNotFound)
}

func promotionVisitorHash(token string) string {
	ctx := service.WithPromotionVisitor(context.Background(), token)
	return service.PromotionVisitorHash(ctx)
}

func isPromotionCrawler(agent string) bool {
	ua := strings.ToLower(agent)
	for _, part := range []string{"bot", "spider", "crawler", "preview", "facebookexternalhit", "twitterbot", "discordbot", "slackbot", "whatsapp", "telegrambot"} {
		if strings.Contains(ua, part) {
			return true
		}
	}
	return false
}
func promotionDevice(agent string) string {
	ua := strings.ToLower(agent)
	if strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet") {
		return "tablet"
	}
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "iphone") || strings.Contains(ua, "android") {
		return "mobile"
	}
	return "desktop"
}

func promotionFilter(c *gin.Context) service.PromotionFilter {
	zone := strings.TrimSpace(c.Query("timezone"))
	if zone == "" {
		zone = timezone.Name()
	}
	loc, err := time.LoadLocation(zone)
	if err != nil || len(zone) > 80 {
		zone = timezone.Name()
		loc = timezone.Location()
	}
	parse := func(key string, fallback time.Time) time.Time {
		value := c.Query(key)
		if value == "" {
			return fallback
		}
		parsed, e := time.ParseInLocation("2006-01-02", value, loc)
		if e != nil {
			return fallback
		}
		return parsed
	}
	now := time.Now().In(loc)
	start := timezone.StartOfDay(now.AddDate(0, 0, -29))
	start = parse("start_at", start)
	end := parse("end_at", timezone.StartOfDay(now)).AddDate(0, 0, 1)
	if !end.After(start) {
		end = start.AddDate(0, 0, 1)
	}
	if end.Sub(start) > 400*24*time.Hour {
		end = start.Add(400 * 24 * time.Hour)
	}
	channel, _ := strconv.ParseInt(c.Query("channel_id"), 10, 64)
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return service.PromotionFilter{Start: start, End: end, Timezone: loc.String(), ChannelID: channel, Search: strings.TrimSpace(c.Query("search")), Status: strings.TrimSpace(c.Query("status")), Page: page, PageSize: size}
}
func (h *PromotionHandler) ListChannels(c *gin.Context) {
	v, e := h.service.Channels(c.Request.Context())
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, v)
}
func (h *PromotionHandler) SaveChannel(c *gin.Context) {
	var v service.PromotionChannel
	if err := c.ShouldBindJSON(&v); err != nil {
		response.BadRequest(c, "invalid channel")
		return
	}
	if raw := c.Param("id"); raw != "" {
		id, e := strconv.ParseInt(raw, 10, 64)
		if e != nil || id <= 0 {
			response.BadRequest(c, "invalid channel id")
			return
		}
		v.ID = id
	}
	if err := h.service.SaveChannel(c.Request.Context(), &v); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}
func (h *PromotionHandler) GetSettings(c *gin.Context) {
	v, e := h.service.Settings(c.Request.Context())
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, v)
}
func (h *PromotionHandler) SaveSettings(c *gin.Context) {
	var v service.PromotionSettings
	if e := c.ShouldBindJSON(&v); e != nil {
		response.BadRequest(c, "invalid settings")
		return
	}
	if e := h.service.SaveSettings(c.Request.Context(), v); e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, v)
}
func (h *PromotionHandler) Report(c *gin.Context) {
	v, e := h.service.Report(c.Request.Context(), promotionFilter(c))
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, json.RawMessage(v))
}
func (h *PromotionHandler) Details(c *gin.Context) {
	v, e := h.service.Details(c.Request.Context(), c.Param("kind"), promotionFilter(c))
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, json.RawMessage(v))
}
func (h *PromotionHandler) AddExpense(c *gin.Context) {
	var v service.PromotionExpense
	if e := c.ShouldBindJSON(&v); e != nil {
		response.BadRequest(c, "invalid expense")
		return
	}
	if e := h.service.AddExpense(c.Request.Context(), v); e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, gin.H{"created": true})
}
func (h *PromotionHandler) DeleteExpense(c *gin.Context) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		response.BadRequest(c, "invalid expense id")
		return
	}
	if e = h.service.DeleteExpense(c.Request.Context(), id); e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
