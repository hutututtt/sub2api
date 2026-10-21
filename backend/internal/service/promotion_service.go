package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type PromotionSettings struct {
	AttributionModel string `json:"attribution_model"`
	WindowDays       int    `json:"window_days"`
}

type PromotionChannel struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Website     string    `json:"website"`
	Placement   string    `json:"placement"`
	LandingPath string    `json:"landing_path"`
	Notes       string    `json:"notes"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

type PromotionVisit struct {
	Code         string `json:"code"`
	EventID      string `json:"event_id"`
	LandingPath  string `json:"landing_path"`
	ReferrerHost string `json:"referrer"`
	Device       string `json:"-"`
}

type PromotionFilter struct {
	Start     time.Time
	End       time.Time // exclusive
	Timezone  string
	ChannelID int64
	Search    string
	Status    string
	Page      int
	PageSize  int
}

type PromotionExpense struct {
	ChannelID int64  `json:"channel_id"`
	SpentOn   string `json:"spent_on"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
	Notes     string `json:"notes"`
}

type PromotionRepository interface {
	Channels(context.Context) ([]PromotionChannel, error)
	SaveChannel(context.Context, *PromotionChannel) error
	Settings(context.Context) (PromotionSettings, error)
	SaveSettings(context.Context, PromotionSettings) error
	Track(context.Context, string, PromotionVisit) (bool, error)
	Report(context.Context, PromotionFilter) (json.RawMessage, error)
	Details(context.Context, string, PromotionFilter) (json.RawMessage, error)
	AddExpense(context.Context, PromotionExpense) error
	DeleteExpense(context.Context, int64) error
}

type PromotionService struct{ repo PromotionRepository }

func NewPromotionService(repo PromotionRepository) *PromotionService {
	return &PromotionService{repo: repo}
}
func (s *PromotionService) Channels(ctx context.Context) ([]PromotionChannel, error) {
	return s.repo.Channels(ctx)
}
func (s *PromotionService) Settings(ctx context.Context) (PromotionSettings, error) {
	return s.repo.Settings(ctx)
}
func (s *PromotionService) SaveSettings(ctx context.Context, v PromotionSettings) error {
	if (v.AttributionModel != "first_touch" && v.AttributionModel != "last_touch") || v.WindowDays < 1 || v.WindowDays > 90 {
		return infraerrors.BadRequest("PROMOTION_INVALID_SETTINGS", "归因期限需为 1–90 天，归因模式需为首次或最近点击")
	}
	return s.repo.SaveSettings(ctx, v)
}

var promotionCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)
var promotionHashPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var promotionAmountPattern = regexp.MustCompile(`^\d{1,12}(\.\d{1,4})?$`)
var promotionCurrencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

func ValidPromotionLanding(path string) bool {
	u, err := url.Parse(path)
	return err == nil && strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") && !strings.ContainsAny(path, "\\\r\n") && !u.IsAbs() && u.Host == "" && len(path) <= 1000
}

func (s *PromotionService) SaveChannel(ctx context.Context, c *PromotionChannel) error {
	c.Name = strings.TrimSpace(c.Name)
	c.Code = strings.TrimSpace(c.Code)
	if !promotionCodePattern.MatchString(c.Code) || c.Name == "" || len([]rune(c.Name)) > 100 || len(c.Website) > 255 || len(c.Placement) > 255 || len(c.Notes) > 4000 || !ValidPromotionLanding(c.LandingPath) {
		return infraerrors.BadRequest("PROMOTION_INVALID_CHANNEL", "请填写渠道名称、3–64 位字母数字标识和站内落地页路径")
	}
	return s.repo.SaveChannel(ctx, c)
}

func (s *PromotionService) Track(ctx context.Context, tokenHash string, v PromotionVisit) (bool, error) {
	if !promotionCodePattern.MatchString(v.Code) || !promotionHashPattern.MatchString(tokenHash) {
		return false, nil
	}
	if !ValidPromotionLanding(v.LandingPath) {
		v.LandingPath = "/"
	}
	if u, err := url.Parse(v.ReferrerHost); err == nil {
		v.ReferrerHost = u.Hostname()
	} else {
		v.ReferrerHost = ""
	}
	if len(v.ReferrerHost) > 255 {
		v.ReferrerHost = ""
	}
	return s.repo.Track(ctx, tokenHash, v)
}
func (s *PromotionService) Report(ctx context.Context, f PromotionFilter) (json.RawMessage, error) {
	return s.repo.Report(ctx, f)
}
func (s *PromotionService) Details(ctx context.Context, kind string, f PromotionFilter) (json.RawMessage, error) {
	return s.repo.Details(ctx, kind, f)
}
func (s *PromotionService) AddExpense(ctx context.Context, v PromotionExpense) error {
	v.Currency = strings.ToUpper(strings.TrimSpace(v.Currency))
	_, dateErr := time.Parse("2006-01-02", v.SpentOn)
	amount, amountErr := strconv.ParseFloat(v.Amount, 64)
	if v.ChannelID <= 0 || dateErr != nil || amountErr != nil || amount <= 0 || !promotionAmountPattern.MatchString(v.Amount) || !promotionCurrencyPattern.MatchString(v.Currency) || len(v.Notes) > 1000 {
		return infraerrors.BadRequest("PROMOTION_INVALID_EXPENSE", "请填写有效的渠道、日期、正数金额和三位币种")
	}
	return s.repo.AddExpense(ctx, v)
}
func (s *PromotionService) DeleteExpense(ctx context.Context, id int64) error {
	return s.repo.DeleteExpense(ctx, id)
}

type promotionVisitorContextKey struct{}

func WithPromotionVisitor(ctx context.Context, token string) context.Context {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != 32 {
		return ctx
	}
	hash := sha256.Sum256(decoded)
	return context.WithValue(ctx, promotionVisitorContextKey{}, hex.EncodeToString(hash[:]))
}
func PromotionVisitorHash(ctx context.Context) string {
	value, _ := ctx.Value(promotionVisitorContextKey{}).(string)
	return value
}
