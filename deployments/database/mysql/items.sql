SET NAMES utf8;
SET time_zone = '+00:00';
SET foreign_key_checks = 0;
SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';

DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` BINARY(16) PRIMARY KEY,
  `username` VARCHAR(32) NOT NULL,
  `hash` CHAR(32) NOT NULL,
  INDEX (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `sessions`;
CREATE TABLE `sessions` (
  `id` varchar(32) NOT NULL,
  `user_id` BINARY(16) NOT NULL,
  UNIQUE KEY `id` (`id`),
  KEY `user_id` (`user_id`),
  FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

DROP TABLE IF EXISTS `clients`;
CREATE TABLE `clients` (
    `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `user_id` BINARY(16) NOT NULL,
    `balance` int NOT NULL,
    UNIQUE KEY `user_id` (`user_id`),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

DROP TABLE IF EXISTS `positions`;
CREATE TABLE `positions` (
    `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `user_id` BINARY(16) NOT NULL,
    `ticker` varchar(300) NOT NULL,
    `volume` int NOT NULL,
    KEY user_id(user_id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

DROP TABLE IF EXISTS `orders_history`;
CREATE TABLE `orders_history` (
    `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `time` int NOT NULL,
    `user_id` BINARY(16) NOT NULL,
    `ticker` varchar(300) NOT NULL,
    `volume` int NOT NULL,
    `price` float not null,
    `is_buy` int not null,
    KEY user_id(user_id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

DROP TABLE IF EXISTS `request`;
CREATE TABLE `request` ( -- запросы
    `id` bigint NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `user_id` BINARY(16) NOT NULL,
    `ticker` varchar(300) NOT NULL,
    `volume` int NOT NULL,
    `price` float NOT NULL,
    `is_buy` int not null, -- 1 - покупаем, 0 - продаем
    KEY user_id(user_id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

DROP TABLE IF EXISTS `stat`;
CREATE TABLE `stat` ( -- запросы
    `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `time` int,
    `interval` int,
    `open` float,
    `high` float,
    `low` float,
    `close` float,
    `volume` int,
    `ticker` varchar(300),
    KEY id(id)
);

INSERT INTO `users`
VALUES (UUID_TO_BIN('177da24e-1e5a-4eee-84c2-8a5fe4457f03'), 'megaurich', '25d55ad283aa400af464c76d713c07ad');

INSERT INTO `clients` (`user_id`, `balance`) VALUES (UUID_TO_BIN('177da24e-1e5a-4eee-84c2-8a5fe4457f03'), 100500000);