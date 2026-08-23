# Transaksi Detail

Skema tabel `transaksi_detail` — menyimpan rincian item per transaksi.

## Table Transaksi Detail

| Kolom             | Tipe         | Unique | Primary Key | Deskripsi                     |
| :---------------- | :----------- | :----: | :---------: | :---------------------------- |
| id                | bigint       |   ✅   |     ✅      | id transaksi detail           |
| transaction_id    | bigint       |   ❌   |     ❌      | id transaksi                   |
| product_name      | varchar(255) |   ❌   |     ❌      | nama produk                   |
| total_product     | varchar(255) |   ❌   |     ❌      | jumlah produk yang dibeli     |
| amount            | varchar(255) |   ❌   |     ❌      | harga per pcs                 |
| total_amount      | varchar(255) |   ❌   |     ❌      | total harga produk            |
| created_at        | timestamp    |   ❌   |     ❌      | tanggal dibuat                |
| updated_at        | timestamp    |   ❌   |     ❌      | tanggal diupdate              |
