# Produk

Skema tabel `produk` — menyimpan data produk yang dijual.

## Table Produk

| Kolom             | Tipe         | Unique | Primary Key | Deskripsi          |
| :---------------- | :----------- | :----: | :---------: | :----------------- |
| id                | bigint       |   ✅   |     ✅      | id produk          |
| name              | varchar(255) |   ❌   |     ❌      | nama produk        |
| stock             | uint16       |   ❌   |     ❌      | stok produk        |
| purchase_price    | decimal(12,2)|   ❌   |     ❌      | harga beli         |
| selling_price     | decimal(12,2)|   ❌   |     ❌      | harga jual         |
| created_at        | timestamp    |   ❌   |     ❌      | tanggal dibuat     |
| updated_at        | timestamp    |   ❌   |     ❌      | tanggal diupdate   |
