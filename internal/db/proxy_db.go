package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

// ── Dashboard ──

func (d *DB) ProxyDashboard(ctx context.Context) (today map[string]interface{}, month map[string]interface{}, daily []map[string]interface{}, topModels interface{}) {
	var tcalls, ttokens int
	var tcost float64
	d.pool.QueryRow(ctx, `SELECT COALESCE(COUNT(*),0), COALESCE(SUM(total_tokens),0), COALESCE(SUM(cost)::numeric,0)
		FROM chat.proxy_usage WHERE created_at::date = CURRENT_DATE`).Scan(&tcalls, &ttokens, &tcost)
	today = map[string]interface{}{"calls": tcalls, "tokens": ttokens, "cost": tcost}

	var mcalls, mtokens int
	var mcost float64
	d.pool.QueryRow(ctx, `SELECT COALESCE(COUNT(*),0), COALESCE(SUM(total_tokens),0), COALESCE(SUM(cost)::numeric,0)
		FROM chat.proxy_usage WHERE date_trunc('month', created_at) = date_trunc('month', CURRENT_DATE)`).Scan(&mcalls, &mtokens, &mcost)
	month = map[string]interface{}{"calls": mcalls, "tokens": mtokens, "cost": mcost}

	daily = []map[string]interface{}{}
	rows, _ := d.pool.Query(ctx, `SELECT created_at::date::text, COUNT(*), SUM(total_tokens), SUM(cost)::numeric
		FROM chat.proxy_usage WHERE created_at >= CURRENT_DATE - 30
		GROUP BY created_at::date ORDER BY created_at::date`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var date string
			var calls, tokens int
			var cost float64
			rows.Scan(&date, &calls, &tokens, &cost)
			daily = append(daily, map[string]interface{}{"date": date, "calls": calls, "tokens": tokens, "cost": cost})
		}
	}
	topModels = nil
	return
}

// ── Models ──

func (d *DB) ProxyListModels(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := d.pool.Query(ctx, `SELECT id, name, display_name, provider, endpoint, api_key, status, priority, max_rpm
		FROM chat.proxy_models ORDER BY priority DESC, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id int
		var name, display, provider, endpoint, apiKey, status string
		var priority, maxRpm int
		rows.Scan(&id, &name, &display, &provider, &endpoint, &apiKey, &status, &priority, &maxRpm)
		out = append(out, map[string]interface{}{
			"id": id, "name": name, "display_name": display, "provider": provider,
			"endpoint": endpoint, "api_key": apiKey, "status": status, "priority": priority, "max_rpm": maxRpm,
		})
	}
	return out, nil
}

func (d *DB) ProxyCreateModel(ctx context.Context, m map[string]interface{}) error {
	_, err := d.pool.Exec(ctx, `INSERT INTO chat.proxy_models (name,display_name,provider,endpoint,api_key,status,priority,max_rpm)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		m["name"], m["display_name"], m["provider"], m["endpoint"], m["api_key"],
		m["status"], m["priority"], m["max_rpm"])
	return err
}

func (d *DB) ProxyUpdateModel(ctx context.Context, id string, m map[string]interface{}) error {
	_, err := d.pool.Exec(ctx, `UPDATE chat.proxy_models SET
		display_name=COALESCE($2,display_name), provider=COALESCE($3,provider),
		endpoint=COALESCE($4,endpoint), api_key=COALESCE($5,api_key),
		status=COALESCE($6,status), priority=COALESCE($7::int,priority), max_rpm=COALESCE($8::int,max_rpm)
		WHERE id=$1::int`,
		id, m["display_name"], m["provider"], m["endpoint"], m["api_key"],
		m["status"], m["priority"], m["max_rpm"])
	return err
}

func (d *DB) ProxyDeleteModel(ctx context.Context, id string) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM chat.proxy_models WHERE id=$1::int`, id)
	return err
}

// ── Pricing ──

func (d *DB) ProxyGetPricing(ctx context.Context) ([]map[string]interface{}, float64, error) {
	var multiplier float64
	d.pool.QueryRow(ctx, `SELECT COALESCE(amount, 2) FROM chat.proxy_pricing WHERE key='pricing_multiplier'`).Scan(&multiplier)

	rows, err := d.pool.Query(ctx, `SELECT key, name, amount, official_amount, active
		FROM chat.proxy_pricing WHERE key <> 'pricing_multiplier' ORDER BY key`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []map[string]interface{}
	for rows.Next() {
		var key, name string
		var amount, official float64
		var active bool
		rows.Scan(&key, &name, &amount, &official, &active)
		items = append(items, map[string]interface{}{
			"key": key, "name": name, "amount": amount, "official_amount": official, "active": active,
		})
	}
	return items, multiplier, nil
}

func (d *DB) ProxyUpdatePricingItem(ctx context.Context, key string, amount *float64, active *bool) error {
	var err error
	if amount != nil && active != nil {
		_, err = d.pool.Exec(ctx, `UPDATE chat.proxy_pricing SET amount=$2, active=$3, updated_at=now() WHERE key=$1`, key, *amount, *active)
	} else if amount != nil {
		_, err = d.pool.Exec(ctx, `UPDATE chat.proxy_pricing SET amount=$2, updated_at=now() WHERE key=$1`, key, *amount)
	} else if active != nil {
		_, err = d.pool.Exec(ctx, `UPDATE chat.proxy_pricing SET active=$2, updated_at=now() WHERE key=$1`, key, *active)
	}
	return err
}

func (d *DB) ProxyUpdateMultiplier(ctx context.Context, multiplier float64) error {
	_, err := d.pool.Exec(ctx, `INSERT INTO chat.proxy_pricing (key, name, amount) VALUES ('pricing_multiplier','全局倍率',$1)
		ON CONFLICT (key) DO UPDATE SET amount=$1, updated_at=now()`, multiplier)
	return err
}

// ── Merchants (deduplicated by org_id) ──

func (d *DB) ProxyListMerchants(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := d.pool.Query(ctx, `SELECT
		l.org_id,
		COALESCE(SUM(CASE WHEN l.direction='in' THEN l.amount_cny ELSE 0 END), 0) -
		COALESCE(SUM(CASE WHEN l.direction='out' THEN l.amount_cny ELSE 0 END), 0) as balance,
		COALESCE(SUM(CASE WHEN l.direction='out' THEN l.amount_cny ELSE 0 END), 0) as total_used,
		(SELECT COUNT(*) FROM chat.proxy_keys k WHERE k.org_id = l.org_id AND k.status='active') as key_count
		FROM platform_finance_ledger l
		WHERE l.biz_type IN ('ai_recharge','ai_consume','ai_model','trial_credit') AND l.status='completed'
		GROUP BY l.org_id
		HAVING COALESCE(SUM(CASE WHEN l.direction='in' THEN l.amount_cny ELSE 0 END), 0) > 0
		ORDER BY balance DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var orgID string
		var balance, used float64
		var keyCount int
		rows.Scan(&orgID, &balance, &used, &keyCount)
		name := "系统管理员"
		d.pool.QueryRow(ctx, `SELECT COALESCE(nickname, username, '系统管理员') FROM org_user WHERE org_id=$1 LIMIT 1`, orgID).Scan(&name)
		out = append(out, map[string]interface{}{
			"org_id": orgID, "name": name, "balance": balance, "total_used": used, "key_count": keyCount,
		})
	}
	return out, nil
}

func (d *DB) ProxyRechargeMerchant(ctx context.Context, orgID string, amount float64) error {
	txID := "rc_" + randomHex(8)
	_, err := d.pool.Exec(ctx, `INSERT INTO platform_finance_ledger (org_id, tx_no, direction, amount, amount_cny, biz_type, tx_time, status)
		VALUES ($1, $2, 'in', $3, $3, 'ai_recharge', now(), 'completed')`, orgID, txID, amount)
	return err
}

func (d *DB) ProxyMerchantLedger(ctx context.Context, orgID string) ([]map[string]interface{}, error) {
	rows, err := d.pool.Query(ctx, `SELECT direction, amount_cny, biz_type, created_at
		FROM platform_finance_ledger WHERE org_id=$1 AND biz_type IN ('ai_recharge','ai_consume','ai_model','trial_credit')
		ORDER BY created_at DESC LIMIT 100`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var dir string
		var amt float64
		var biz string
		var ts time.Time
		rows.Scan(&dir, &amt, &biz, &ts)
		out = append(out, map[string]interface{}{"direction": dir, "amount_cny": amt, "biz_type": biz, "created_at": ts.Format(time.RFC3339)})
	}
	return out, nil
}

// ── Keys ──

func (d *DB) ProxyListKeys(ctx context.Context, orgID string) ([]map[string]interface{}, error) {
	query := `SELECT id, key_text, org_id, COALESCE(device_id,''), COALESCE(name,''), status, created_at FROM chat.proxy_keys`
	var args []interface{}
	if orgID != "" {
		query += ` WHERE org_id = $1`
		args = append(args, orgID)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := d.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id int
		var keyText, org, devID, name, status string
		var ts time.Time
		rows.Scan(&id, &keyText, &org, &devID, &name, &status, &ts)
		out = append(out, map[string]interface{}{
			"id": id, "key_text": keyText, "org_id": org, "device_id": devID,
			"name": name, "status": status, "created_at": ts.Format(time.RFC3339),
		})
	}
	return out, nil
}

func (d *DB) ProxyCreateKey(ctx context.Context, orgID, name string) (map[string]interface{}, error) {
	keyText := "sk-fz-" + randomHex(48)
	var id int
	err := d.pool.QueryRow(ctx, `INSERT INTO chat.proxy_keys (key_text, org_id, name, status)
		VALUES ($1, $2, $3, 'active') RETURNING id`, keyText, orgID, name).Scan(&id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id, "key_text": keyText, "org_id": orgID, "name": name, "status": "active"}, nil
}

func (d *DB) ProxyRevokeKey(ctx context.Context, id string) error {
	_, err := d.pool.Exec(ctx, `UPDATE chat.proxy_keys SET status='revoked' WHERE id=$1::int`, id)
	return err
}

func randomHex(n int) string {
	b := make([]byte, n/2)
	rand.Read(b)
	return hex.EncodeToString(b)
}
