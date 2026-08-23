# Transaksi

Skema tabel `transaksi` — menyimpan data transaksi penjualan kasir.

## Table Transaksi

| Kolom             | Tipe         | Unique | Primary Key | Deskripsi          |
| :---------------- | :----------- | :----: | :---------: | :----------------- |
| id                | bigint       |   ✅   |     ✅      | id transaksi       |
| kasir             | varchar(255) |   ❌   |     ❌      | username kasir     |
| total_transaction | varchar(255) |   ❌   |     ❌      | total bayar        |
| created_at        | timestamp    |   ❌   |     ❌      | tanggal dibuat     |
| updated_at        | timestamp    |   ❌   |     ❌      | tanggal diupdate   |
