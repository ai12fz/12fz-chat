package db

import "context"

func (d *DB) GetDeviceModelConfig(ctx context.Context, deviceID string) (modelName, modelProvider string, err error) {
	err = d.pool.QueryRow(ctx,
		`SELECT COALESCE(model_name, 'deepseek-v4-flash'), COALESCE(model_provider, 'custom:deepseek-v4-flash(12fz)')
		 FROM chat.devices WHERE id=$1`, deviceID).Scan(&modelName, &modelProvider)
	return
}

func (d *DB) SetDeviceModelConfig(ctx context.Context, deviceID, modelName, modelProvider string) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE chat.devices SET model_name=$2, model_provider=$3 WHERE id=$1`,
		deviceID, modelName, modelProvider)
	return err
}
