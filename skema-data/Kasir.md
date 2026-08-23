# Kasir

Skema tabel `kasir` — menyimpan data akun kasir.

## Table Kasir

| Kolom             | Tipe         | Unique | Primary Key | Deskripsi          |
| :---------------- | :----------- | :----: | :---------: | :----------------- |
| id                | bigint       |   ✅   |     ✅      | id kasir           |
| username          | varchar(255) |   ✅   |     ❌      | username kasir     |
| password          | varchar(255) |   ❌   |     ❌      | sandi kasir        |
| created_at        | timestamp    |   ❌   |     ❌      | tanggal dibuat     |
| updated_at        | timestamp    |   ❌   |     ❌      | tanggal diupdate   |
