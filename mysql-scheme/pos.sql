-- Skema database MySQL untuk aplikasi POS
-- Dibuat berdasarkan skema-data: Kasir.md, Produk.md, Transaksi.md, Transaksi_Detail.md

CREATE DATABASE IF NOT EXISTS `pos`
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE `pos`;

-- ---------------------------------------------------------------------------
-- Table kasir
-- menyimpan data akun kasir
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `kasir` (
  `id`         BIGINT       NOT NULL AUTO_INCREMENT,
  `username`   VARCHAR(255) NOT NULL,
  `password`   VARCHAR(255) NOT NULL,
  `created_at` TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_kasir_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------------
-- Table produk
-- menyimpan data produk yang dijual
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `produk` (
  `id`         BIGINT           NOT NULL AUTO_INCREMENT,
  `name`       VARCHAR(255)     NOT NULL,
  `purchase_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00,
  `selling_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00,
  `stock`      SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------------
-- Table transaksi
-- menyimpan data transaksi penjualan kasir
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `transaksi` (
  `id`                BIGINT       NOT NULL AUTO_INCREMENT,
  `kasir`             VARCHAR(255) NOT NULL,
  `total_transaction` DECIMAL(12,2) NOT NULL DEFAULT 0.00,
  `profit` DECIMAL(12,2) NOT NULL DEFAULT 0.00,
  `created_at`        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------------
-- Table transaksi_detail
-- menyimpan rincian item per transaksi
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `transaksi_detail` (
  `id`            BIGINT       NOT NULL AUTO_INCREMENT,
  `transaction_id`  BIGINT NOT NULL,
  `product_name`  VARCHAR(255) NOT NULL,
  `total_product` VARCHAR(255) NOT NULL,
  `amount`        DECIMAL(12,2) NOT NULL DEFAULT 0,
  `total_amount`  DECIMAL(12,2) NOT NULL DEFAULT 0,
  `created_at`    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
