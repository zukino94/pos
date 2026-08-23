# Produk

Skema tabel `produk` — menyimpan data produk yang dijual.

## Table Produk

| Kolom             | Tipe         | Unique | Primary Key | Deskripsi          |
| :---------------- | :----------- | :----: | :---------: | :----------------- |
| id                | bigint       |   ✅   |     ✅      | id produk          |
| name              | varchar(255) |   ❌   |     ❌      | nama produk        |
| stock             | uint16       |   ❌   |     ❌      | stok produk        |
| created_at        | timestamp    |   ❌   |     ❌      | tanggal dibuat     |
| updated_at        | timestamp    |   ❌   |     ❌      | tanggal diupdate   |
