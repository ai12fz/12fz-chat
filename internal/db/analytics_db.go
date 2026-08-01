package db

import (
	"context"
	"encoding/json"
	"time"
)

// EnsureAnalyticsTable creates analytics_events in the chat DB (12fzsj).
// The analytics service was merged into chat-token; the old service used the
// zhongtai DB, but only test rows existed, so we start fresh here.
func (d *DB) EnsureAnalyticsTable(ctx context.Context) error {
	sql := `CREATE TABLE IF NOT EXISTS analytics_events (
		id BIGSERIAL PRIMARY KEY, app VARCHAR(20) DEFAULT 'web',
		event VARCHAR(32) NOT NULL, page VARCHAR(255),
		title VARCHAR(255), source VARCHAR(255),
		source_type VARCHAR(20), user_id VARCHAR(64),
		session_id VARCHAR(64), ip VARCHAR(45),
		region VARCHAR(32), country VARCHAR(32),
		extra JSONB DEFAULT '{}',
		created_at TIMESTAMP DEFAULT NOW()
	);`
	if _, err := d.pool.Exec(ctx, sql); err != nil {
		return err
	}
	_, err := d.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_ae_date ON analytics_events(created_at); CREATE INDEX IF NOT EXISTS idx_ae_event ON analytics_events(event);`)
	return err
}

// InsertAnalyticsEvent records a tracking event.
func (d *DB) InsertAnalyticsEvent(ctx context.Context, app, event, page, title, source, sourceType, userID, sessionID, ip string, extra map[string]interface{}) error {
	extraJSON, _ := json.Marshal(extra)
	_, err := d.pool.Exec(ctx,
		`INSERT INTO analytics_events(app,event,page,title,source,source_type,user_id,session_id,ip,extra)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		app, event, page, title, source, sourceType, userID, sessionID, ip, string(extraJSON))
	return err
}

// AnalyticsOverview returns PV/UV/tool-use counts plus source and page stats.
func (d *DB) AnalyticsOverview(ctx context.Context, days int) (map[string]interface{}, error) {
	interval := time.Duration(days) * 24 * time.Hour

	var pv, uv, tools int
	if err := d.pool.QueryRow(ctx,
		`SELECT COUNT(*)::int, COUNT(DISTINCT session_id)::int,
		        COUNT(*) FILTER (WHERE event='tool_use')::int
		 FROM analytics_events WHERE created_at > NOW() - $1::interval`,
		interval).Scan(&pv, &uv, &tools); err != nil {
		return nil, err
	}

	rows, err := d.pool.Query(ctx,
		`SELECT source_type, COUNT(*)::int FROM analytics_events
		 WHERE created_at > NOW() - $1::interval AND source_type IS NOT NULL
		 GROUP BY source_type ORDER BY 2 DESC LIMIT 10`, interval)
	if err != nil {
		return nil, err
	}
	var sources []map[string]interface{}
	for rows.Next() {
		var st string
		var cnt int
		if err := rows.Scan(&st, &cnt); err != nil {
			rows.Close()
			return nil, err
		}
		sources = append(sources, map[string]interface{}{"type": st, "count": cnt})
	}
	rows.Close()

	rows, err = d.pool.Query(ctx,
		`SELECT page, COUNT(*)::int AS views, COUNT(DISTINCT session_id)::int AS uv
		 FROM analytics_events WHERE created_at > NOW() - $1::interval AND page IS NOT NULL
		 GROUP BY page ORDER BY views DESC LIMIT 10`, interval)
	if err != nil {
		return nil, err
	}
	var pages []map[string]interface{}
	for rows.Next() {
		var pg string
		var views, puv int
		if err := rows.Scan(&pg, &views, &puv); err != nil {
			rows.Close()
			return nil, err
		}
		pages = append(pages, map[string]interface{}{"page": pg, "views": views, "uv": puv})
	}
	rows.Close()

	return map[string]interface{}{
		"overview":  map[string]interface{}{"pv": pv, "uv": uv, "tools": tools, "register": 0},
		"sources":   sources,
		"top_pages": pages,
		"regions":   []interface{}{},
	}, nil
}
