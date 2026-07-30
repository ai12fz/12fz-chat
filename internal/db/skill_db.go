package db

import "context"

type Skill struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Icon        string `json:"icon"`
	InstallCmd  string `json:"install_cmd"`
	ToolDef     string `json:"tool_definition"`
	Handler     string `json:"handler"`
}

func (db *DB) ListSkills(ctx context.Context, orgID string) ([]map[string]interface{}, error) {
	rows, err := db.pool.Query(ctx, `SELECT name,display_name,description,category,icon,install_cmd,tool_definition::text,handler FROM chat.skills WHERE status='active' AND (org_id = $1 OR org_id = '00000000-0000-0000-0000-000000000000') ORDER BY category,id`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var name, display, desc, cat, icon, install, toolDef, handler string
		rows.Scan(&name, &display, &desc, &cat, &icon, &install, &toolDef, &handler)
		out = append(out, map[string]interface{}{
			"name": name, "display_name": display, "description": desc,
			"category": cat, "icon": icon, "install_cmd": install,
			"tool_definition": toolDef, "handler": handler,
		})
	}
	return out, nil
}

func (db *DB) CreateSkill(ctx context.Context, s map[string]interface{}) error {
	_, err := db.pool.Exec(ctx, `INSERT INTO chat.skills (name,display_name,description,category,icon,install_cmd,tool_definition,handler) VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8)`,
		s["name"], s["display_name"], s["description"], s["category"], s["icon"], s["install_cmd"], s["tool_definition"], s["handler"])
	return err
}

func (db *DB) UpdateSkill(ctx context.Context, name string, s map[string]interface{}) error {
	_, err := db.pool.Exec(ctx, `UPDATE chat.skills SET display_name=$1,description=$2,category=$3,icon=$4,install_cmd=$5,tool_definition=$6::jsonb,handler=$7 WHERE name=$8`,
		s["display_name"], s["description"], s["category"], s["icon"], s["install_cmd"], s["tool_definition"], s["handler"], name)
	return err
}

func (db *DB) DeleteSkill(ctx context.Context, name string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM chat.skills WHERE name=$1`, name)
	return err
}

func (db *DB) ToggleDeviceSkills(ctx context.Context, deviceID string) error {
        _, err := db.pool.Exec(ctx, "UPDATE chat.devices SET allow_install_skills = NOT allow_install_skills WHERE id=$1", deviceID)
        return err
}

func (db *DB) ToggleDeviceSoftware(ctx context.Context, deviceID string) error {
        _, err := db.pool.Exec(ctx, "UPDATE chat.devices SET allow_install_software = NOT allow_install_software WHERE id=$1", deviceID)
        return err
}

// ListCapabilities returns tools + skills for an org
func (db *DB) ListCapabilities(ctx context.Context, orgID string) ([]map[string]interface{}, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, type, name, icon, description FROM chat.capabilities
		 WHERE org_id=$1 OR org_id=$2
		 ORDER BY type, id`, orgID, "00000000-0000-0000-0000-000000000000")
	if err != nil { return nil, err }
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id int; var typ, name, icon, desc string
		if err := rows.Scan(&id, &typ, &name, &icon, &desc); err != nil { continue }
		out = append(out, map[string]interface{}{
			"id": id, "type": typ, "name": name, "icon": icon, "description": desc,
		})
	}
	return out, nil
}
