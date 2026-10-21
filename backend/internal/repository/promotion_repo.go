package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type promotionRepository struct{ db *sql.DB }

func NewPromotionRepository(db *sql.DB) service.PromotionRepository {
	return &promotionRepository{db: db}
}

func (r *promotionRepository) Channels(ctx context.Context) ([]service.PromotionChannel, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,code,name,website,placement,landing_path,notes,enabled,created_at FROM promotion_channels ORDER BY created_at DESC,id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]service.PromotionChannel, 0)
	for rows.Next() {
		var v service.PromotionChannel
		if err = rows.Scan(&v.ID, &v.Code, &v.Name, &v.Website, &v.Placement, &v.LandingPath, &v.Notes, &v.Enabled, &v.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}
func (r *promotionRepository) SaveChannel(ctx context.Context, c *service.PromotionChannel) error {
	var err error
	if c.ID == 0 {
		err = r.db.QueryRowContext(ctx, `INSERT INTO promotion_channels(code,name,website,placement,landing_path,notes,enabled) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,created_at`, c.Code, c.Name, c.Website, c.Placement, c.LandingPath, c.Notes, c.Enabled).Scan(&c.ID, &c.CreatedAt)
	} else {
		// Codes are permanent so existing advertising links and history stay valid.
		err = r.db.QueryRowContext(ctx, `UPDATE promotion_channels SET name=$2,website=$3,placement=$4,landing_path=$5,notes=$6,enabled=$7,updated_at=NOW() WHERE id=$1 AND code=$8 RETURNING created_at`, c.ID, c.Name, c.Website, c.Placement, c.LandingPath, c.Notes, c.Enabled, c.Code).Scan(&c.CreatedAt)
	}
	var pe *pq.Error
	if errors.As(err, &pe) && pe.Code == "23505" {
		return infraerrors.Conflict("PROMOTION_CODE_TAKEN", "渠道标识已存在")
	}
	if errors.Is(err, sql.ErrNoRows) {
		return infraerrors.NotFound("PROMOTION_CHANNEL_NOT_FOUND", "渠道不存在或标识已变更")
	}
	return err
}
func (r *promotionRepository) Settings(ctx context.Context) (service.PromotionSettings, error) {
	var v service.PromotionSettings
	err := r.db.QueryRowContext(ctx, `SELECT attribution_model,window_days FROM promotion_settings WHERE id=1`).Scan(&v.AttributionModel, &v.WindowDays)
	return v, err
}
func (r *promotionRepository) SaveSettings(ctx context.Context, v service.PromotionSettings) error {
	_, err := r.db.ExecContext(ctx, `UPDATE promotion_settings SET attribution_model=$1,window_days=$2 WHERE id=1`, v.AttributionModel, v.WindowDays)
	return err
}

func (r *promotionRepository) Track(ctx context.Context, hash string, v service.PromotionVisit) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var channelID, visitorID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM promotion_channels WHERE code=$1 AND enabled`, v.Code).Scan(&channelID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// The visitor row lock serializes concurrent tabs, making effective clicks reliable.
	err = tx.QueryRowContext(ctx, `INSERT INTO promotion_visitors(token_hash) VALUES($1) ON CONFLICT(token_hash) DO UPDATE SET token_hash=EXCLUDED.token_hash RETURNING id`, hash).Scan(&visitorID)
	if err != nil {
		return false, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO promotion_visits(event_id,visitor_id,channel_id,is_effective,referrer_host,landing_path,device)
	SELECT $1,$2,$3,NOT EXISTS(SELECT 1 FROM promotion_visits WHERE visitor_id=$2 AND channel_id=$3 AND is_effective AND created_at>NOW()-INTERVAL '30 minutes'),$4,$5,$6
	ON CONFLICT(event_id) DO NOTHING`, v.EventID, visitorID, channelID, v.ReferrerHost, v.LandingPath, v.Device)
	if err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// Called inside the same Ent transaction that creates the user. Existing logins
// never invoke this hook, and a failed registration cannot leave an attribution.
func attachPromotionAttribution(ctx context.Context, exec sqlQueryExecutor, userID int64, hash string) error {
	_, err := exec.ExecContext(ctx, `WITH eligible AS (
	 SELECT v.*,s.attribution_model,s.window_days FROM promotion_visits v
	 JOIN promotion_visitors visitor ON visitor.id=v.visitor_id
	 CROSS JOIN promotion_settings s
	 WHERE visitor.token_hash=$2 AND v.is_effective AND v.created_at>=NOW()-make_interval(days=>s.window_days)
	), first_visit AS (SELECT * FROM eligible ORDER BY created_at,id LIMIT 1),
	last_visit AS (SELECT * FROM eligible ORDER BY created_at DESC,id DESC LIMIT 1)
	INSERT INTO promotion_attributions(user_id,visitor_id,channel_id,first_channel_id,last_channel_id,visit_id,attribution_model,window_days)
	SELECT $1,l.visitor_id,CASE WHEN l.attribution_model='first_touch' THEN f.channel_id ELSE l.channel_id END,
	f.channel_id,l.channel_id,CASE WHEN l.attribution_model='first_touch' THEN f.id ELSE l.id END,l.attribution_model,l.window_days
	FROM first_visit f CROSS JOIN last_visit l ON CONFLICT(user_id) DO NOTHING`, userID, hash)
	return err
}

// Receipt amounts are real payment currency; balance credits and gifts are USD.
// refund_amount is a USD credit amount, so convert with the same order ratio as
// PaymentService, only after a successful refund (requested/pending != refunded).
const promotionReceiptsCTE = `receipts AS (
	SELECT o.*,a.channel_id,a.visitor_id,a.registered_at,a.window_days,
 COALESCE(NULLIF(UPPER(o.provider_snapshot->>'currency'),''),'CNY') AS currency,
 CASE WHEN o.status IN ('REFUNDED','PARTIALLY_REFUNDED') AND o.refund_at IS NOT NULL AND o.amount>0
 THEN ROUND(o.pay_amount*LEAST(GREATEST(o.refund_amount,0),o.amount)/o.amount,
 CASE WHEN UPPER(o.provider_snapshot->>'currency') IN ('JPY','KRW','VND','CLP') THEN 0
 WHEN UPPER(o.provider_snapshot->>'currency') IN ('KWD','BHD','OMR','TND') THEN 3 ELSE 2 END)
 ELSE 0 END AS cash_refund
 FROM payment_orders o JOIN promotion_attributions a ON a.user_id=o.user_id
 WHERE o.paid_at IS NOT NULL AND o.order_type='balance' AND ($3::bigint=0 OR a.channel_id=$3)
)`

func (r *promotionRepository) Report(ctx context.Context, f service.PromotionFilter) (json.RawMessage, error) {
	query := `WITH ` + promotionReceiptsCTE + `,
 visits AS (SELECT * FROM promotion_visits WHERE created_at >= $1 AND created_at < $2 AND ($3::bigint=0 OR channel_id=$3)),
	cohort AS (SELECT DISTINCT channel_id,visitor_id FROM visits),
	converted AS (SELECT DISTINCT a.channel_id,a.visitor_id FROM promotion_attributions a JOIN cohort c
	 ON c.channel_id=a.channel_id AND c.visitor_id=a.visitor_id
	 WHERE EXISTS (SELECT 1 FROM visits v WHERE v.channel_id=a.channel_id AND v.visitor_id=a.visitor_id
	   AND a.registered_at>=v.created_at AND a.registered_at<=v.created_at+make_interval(days=>a.window_days))),
 registrations AS (SELECT * FROM promotion_attributions WHERE registered_at >= $1 AND registered_at < $2 AND ($3::bigint=0 OR channel_id=$3)),
 scopes AS (SELECT 0::bigint AS id,'全部渠道'::text AS name,''::text AS code,true AS enabled UNION ALL
 SELECT id,name,code,enabled FROM promotion_channels WHERE $3::bigint=0 OR id=$3),
 metrics AS (SELECT s.*,jsonb_build_object(
 'channel_id',s.id,'name',s.name,'code',s.code,'enabled',s.enabled,
 'clicks',(SELECT COUNT(*) FROM visits v WHERE s.id=0 OR v.channel_id=s.id),
 'effective_clicks',(SELECT COUNT(*) FROM visits v WHERE v.is_effective AND (s.id=0 OR v.channel_id=s.id)),
 'visitors',(SELECT COUNT(DISTINCT visitor_id) FROM cohort v WHERE s.id=0 OR v.channel_id=s.id),
 'converted_visitors',(SELECT COUNT(DISTINCT visitor_id) FROM converted v WHERE s.id=0 OR v.channel_id=s.id),
 'converted_payers',(SELECT COUNT(DISTINCT v.visitor_id) FROM converted v WHERE (s.id=0 OR v.channel_id=s.id) AND EXISTS(SELECT 1 FROM receipts p WHERE p.visitor_id=v.visitor_id AND p.channel_id=v.channel_id)),
 'registrations',(SELECT COUNT(*) FROM registrations v WHERE s.id=0 OR v.channel_id=s.id),
 'paying_users',(SELECT COUNT(DISTINCT user_id) FROM receipts p WHERE paid_at >= $1 AND paid_at < $2 AND (s.id=0 OR p.channel_id=s.id)),
 'orders',(SELECT COUNT(*) FROM receipts p WHERE paid_at >= $1 AND paid_at < $2 AND (s.id=0 OR p.channel_id=s.id)),
 'cohort_payers',(SELECT COUNT(DISTINCT p.user_id) FROM receipts p JOIN registrations a ON a.user_id=p.user_id WHERE s.id=0 OR p.channel_id=s.id),
 'amounts',COALESCE((SELECT jsonb_agg(to_jsonb(m)) FROM (
   SELECT currency,SUM(pay_amount) AS paid,SUM(cash_refund) AS refunded,SUM(pay_amount-cash_refund) AS net,
   COUNT(*) AS orders,COUNT(DISTINCT user_id) AS payers,SUM(bonus_amount) AS bonus_usd
   FROM receipts p WHERE paid_at >= $1 AND paid_at < $2 AND (s.id=0 OR p.channel_id=s.id) GROUP BY currency
 ) m),'[]'::jsonb),
 'cohort_amounts',COALESCE((SELECT jsonb_agg(to_jsonb(m)) FROM (
   SELECT currency,SUM(pay_amount) AS paid,SUM(cash_refund) AS refunded,SUM(pay_amount-cash_refund) AS net
   FROM receipts p JOIN registrations a ON a.user_id=p.user_id WHERE s.id=0 OR p.channel_id=s.id GROUP BY currency
 ) m),'[]'::jsonb),
 'expenses',COALESCE((SELECT jsonb_agg(to_jsonb(m)) FROM (
   SELECT currency,SUM(amount) AS amount FROM promotion_expenses e WHERE spent_on >= ($1::timestamptz AT TIME ZONE $4)::date
   AND spent_on < ($2::timestamptz AT TIME ZONE $4)::date AND (s.id=0 OR e.channel_id=s.id) GROUP BY currency
 ) m),'[]'::jsonb)
 ) AS data FROM scopes s),
 days AS (SELECT generate_series(($1::timestamptz AT TIME ZONE $4)::date,(($2::timestamptz AT TIME ZONE $4)::date-1),INTERVAL '1 day')::date AS day)
 SELECT jsonb_build_object('summary',(SELECT data FROM metrics WHERE id=0),
 'channels',COALESCE((SELECT jsonb_agg(data ORDER BY (data->>'registrations')::bigint DESC,id) FROM metrics WHERE id<>0),'[]'::jsonb),
 'trend',COALESCE((SELECT jsonb_agg(jsonb_build_object('date',to_char(day,'YYYY-MM-DD'),
 'clicks',(SELECT COUNT(*) FROM visits WHERE (created_at AT TIME ZONE $4)::date=day),
 'visitors',(SELECT COUNT(DISTINCT visitor_id) FROM visits WHERE (created_at AT TIME ZONE $4)::date=day),
 'registrations',(SELECT COUNT(*) FROM registrations WHERE (registered_at AT TIME ZONE $4)::date=day)) ORDER BY day) FROM days),'[]'::jsonb))`
	var data []byte
	err := r.db.QueryRowContext(ctx, query, f.Start, f.End, f.ChannelID, f.Timezone).Scan(&data)
	return json.RawMessage(data), err
}

func (r *promotionRepository) Details(ctx context.Context, kind string, f service.PromotionFilter) (json.RawMessage, error) {
	var source string
	switch kind {
	case "users":
		source = `SELECT a.user_id AS id,a.user_id,a.channel_id,c.name AS channel_name,u.email,u.username,u.status AS user_status,
 a.registered_at AS created_at,u.last_active_at,a.attribution_model,fc.name AS first_channel,lc.name AS last_channel,
 (SELECT MIN(paid_at) FROM receipts WHERE user_id=a.user_id) AS first_paid_at,
 (SELECT COUNT(*) FROM receipts WHERE user_id=a.user_id) AS orders,
 COALESCE((SELECT jsonb_agg(to_jsonb(m)) FROM (SELECT currency,SUM(pay_amount) AS paid,SUM(cash_refund) AS refunded,SUM(pay_amount-cash_refund) AS net FROM receipts WHERE user_id=a.user_id GROUP BY currency)m),'[]'::jsonb) AS amounts,
 CASE WHEN EXISTS(SELECT 1 FROM receipts WHERE user_id=a.user_id) THEN 'paid' ELSE 'registered' END AS status
 FROM promotion_attributions a JOIN users u ON u.id=a.user_id JOIN promotion_channels c ON c.id=a.channel_id
 JOIN promotion_channels fc ON fc.id=a.first_channel_id JOIN promotion_channels lc ON lc.id=a.last_channel_id
 WHERE a.registered_at >= $1 AND a.registered_at < $2 AND ($3::bigint=0 OR a.channel_id=$3)
 AND ($4='' OR u.email ILIKE '%'||$4||'%' OR u.username ILIKE '%'||$4||'%' OR a.user_id::text=$4)`
	case "visitors":
		source = `SELECT MIN(v.id) AS id,v.channel_id,c.name AS channel_name,v.visitor_id,
	 'V-'||v.visitor_id::text AS visitor_label,vc.first_visit_at AS created_at,MAX(v.created_at) AS last_visit_at,
 COUNT(*) AS clicks,COUNT(*) FILTER(WHERE v.is_effective) AS effective_clicks,
 (array_agg(v.device ORDER BY v.created_at DESC,v.id DESC))[1] AS device,
 (array_agg(v.referrer_host ORDER BY v.created_at DESC,v.id DESC))[1] AS referrer_host,
	 (SELECT COUNT(*) FROM promotion_attributions a WHERE a.visitor_id=v.visitor_id AND a.channel_id=v.channel_id
	   AND EXISTS(SELECT 1 FROM visits cv WHERE cv.visitor_id=v.visitor_id AND cv.channel_id=v.channel_id
	     AND a.registered_at>=cv.created_at AND a.registered_at<=cv.created_at+make_interval(days=>a.window_days))) AS registrations,
	 CASE WHEN EXISTS(SELECT 1 FROM receipts p WHERE p.visitor_id=v.visitor_id AND p.channel_id=v.channel_id
	   AND EXISTS(SELECT 1 FROM visits cv WHERE cv.visitor_id=v.visitor_id AND cv.channel_id=v.channel_id
	     AND p.registered_at>=cv.created_at AND p.registered_at<=cv.created_at+make_interval(days=>p.window_days))) THEN 'paid'
	 WHEN EXISTS(SELECT 1 FROM promotion_attributions a WHERE a.visitor_id=v.visitor_id AND a.channel_id=v.channel_id
	   AND EXISTS(SELECT 1 FROM visits cv WHERE cv.visitor_id=v.visitor_id AND cv.channel_id=v.channel_id
	     AND a.registered_at>=cv.created_at AND a.registered_at<=cv.created_at+make_interval(days=>a.window_days))) THEN 'registered' ELSE 'unregistered' END AS status
	 FROM visits v JOIN visitor_cohorts vc ON vc.visitor_id=v.visitor_id AND vc.channel_id=v.channel_id
	 JOIN promotion_channels c ON c.id=v.channel_id
	 WHERE ($4='' OR v.visitor_id::text=$4 OR v.referrer_host ILIKE '%'||$4||'%') GROUP BY v.channel_id,c.name,v.visitor_id,vc.first_visit_at`
	case "payments":
		source = `SELECT p.id,p.user_id,p.channel_id,c.name AS channel_name,u.email,u.username,p.paid_at AS created_at,
 p.pay_amount,p.currency,p.cash_refund,p.pay_amount-p.cash_refund AS net,p.bonus_amount AS bonus_usd,p.status,p.payment_type,p.out_trade_no
 FROM receipts p JOIN users u ON u.id=p.user_id JOIN promotion_channels c ON c.id=p.channel_id
 WHERE p.paid_at >= $1 AND p.paid_at < $2 AND ($4='' OR u.email ILIKE '%'||$4||'%' OR p.out_trade_no ILIKE '%'||$4||'%' OR p.user_id::text=$4)`
	case "expenses":
		source = `SELECT e.id,e.channel_id,c.name AS channel_name,e.spent_on AS created_at,e.amount,e.currency,e.notes
 FROM promotion_expenses e JOIN promotion_channels c ON c.id=e.channel_id
 WHERE e.spent_on>=($1::timestamptz AT TIME ZONE $8)::date AND e.spent_on<($2::timestamptz AT TIME ZONE $8)::date
 AND ($3::bigint=0 OR e.channel_id=$3) AND ($4='' OR e.notes ILIKE '%'||$4||'%')`
	default:
		return nil, infraerrors.BadRequest("PROMOTION_INVALID_DETAIL", "未知的明细类型")
	}
	// All identifiers are chosen above. Only values come from query parameters.
	hasStatusFilter := kind == "users" || kind == "visitors"
	statusClause := ""
	if hasStatusFilter {
		statusClause = " WHERE ($5='' OR status=$5)"
	}
	query := `WITH ` + promotionReceiptsCTE + `, visits AS (SELECT * FROM promotion_visits WHERE created_at >= $1 AND created_at < $2 AND ($3::bigint=0 OR channel_id=$3)), visitor_cohorts AS (SELECT visitor_id,channel_id,MIN(created_at) AS first_visit_at FROM visits GROUP BY visitor_id,channel_id), detail AS (` + source + `), filtered AS (SELECT * FROM detail` + statusClause + `)
 SELECT jsonb_build_object('items',COALESCE((SELECT jsonb_agg(to_jsonb(p)) FROM (SELECT * FROM filtered ORDER BY created_at DESC,id DESC LIMIT $6 OFFSET $7)p),'[]'::jsonb),
 'total',(SELECT COUNT(*) FROM filtered),'page',$9::int,'page_size',$6::int,'timezone',$8::text)`
	var data []byte
	args := []any{f.Start, f.End, f.ChannelID, f.Search}
	if hasStatusFilter {
		args = append(args, f.Status)
	} else {
		// Non-status detail queries omit $5, so shift pagination and timezone
		// placeholders down to keep PostgreSQL's bind parameter positions dense.
		query = strings.NewReplacer("$9", "$8", "$8", "$7", "$7", "$6", "$6", "$5").Replace(query)
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize, f.Timezone, f.Page)
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&data)
	return json.RawMessage(data), err
}

func (r *promotionRepository) AddExpense(ctx context.Context, v service.PromotionExpense) error {
	result, err := r.db.ExecContext(ctx, `INSERT INTO promotion_expenses(channel_id,spent_on,amount,currency,notes) SELECT id,$2::date,$3::numeric,$4,$5 FROM promotion_channels WHERE id=$1 AND $3::numeric>0`, v.ChannelID, v.SpentOn, v.Amount, v.Currency, v.Notes)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return infraerrors.BadRequest("PROMOTION_INVALID_EXPENSE", "渠道不存在或金额需大于零")
	}
	return err
}
func (r *promotionRepository) DeleteExpense(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM promotion_expenses WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return infraerrors.NotFound("PROMOTION_EXPENSE_NOT_FOUND", "费用记录不存在")
	}
	return err
}
