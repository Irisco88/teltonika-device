package clickhouse

import (
	"context"
)

const insertRawDataQuery = `
	INSERT INTO rawdatas (imei, payload)
VALUES (?,?);

`

// SaveRawData saves raw data to clickhouse
func (adb *AVLDataBase) SaveRawData(ctx context.Context, imei, payload string) error {
	batch, err := adb.GetConn().PrepareBatch(ctx, insertRawDataQuery)
	if err != nil {
		return err
	}
	if e := batch.Append(imei, payload); e != nil {
		return e
	}
	return batch.Send()
}
