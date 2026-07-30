package db

import "context"

func (d *DB) LookupProxyKey(ctx context.Context, keyText string) (int, string, error) {
	var id int
	var orgID string
	err := d.pool.QueryRow(ctx,
		`SELECT id, org_id FROM chat.proxy_keys WHERE key_text=$1 AND status='active'`, keyText).Scan(&id, &orgID)
	if err != nil {
		return 0, "", err
	}
	return id, orgID, nil
}
