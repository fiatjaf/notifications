package main

func setLastTelegramUpdate(ltu int64) error {
	_, err := pg.Exec(`
      UPDATE kv
      SET value = to_jsonb($1)
      WHERE key = 'last_telegram_update'
    `, ltu)
	return err
}

func getLastTelegramUpdate() (ltu int64, err error) {
	err = pg.Get(&ltu, `
      SELECT (jsonb_build_object('value', value)->>'value')::int
      FROM kv
      WHERE key = 'last_telegram_update'
    `)
	return
}
