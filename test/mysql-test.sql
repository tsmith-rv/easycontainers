CREATE DATABASE IF NOT EXISTS `blog`;

CREATE TABLE `blog`.`authors` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `first_name` varchar(50) COLLATE utf8_unicode_ci NOT NULL,
  `last_name` varchar(50) COLLATE utf8_unicode_ci NOT NULL,
  `email` varchar(100) COLLATE utf8_unicode_ci NOT NULL,
  `birthdate` varchar(50) COLLATE utf8_unicode_ci NOT NULL,
  `added` varchar(50) COLLATE utf8_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `email` (`email`)
) ENGINE=InnoDB AUTO_INCREMENT=101 DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

INSERT INTO `blog`.`authors` (`id`, `first_name`, `last_name`, `email`, `birthdate`, `added`) VALUES (1, 'Terrill', 'Buckridge', 'zmcglynn@example.org', '1989-03-30', '1976-06-06 21:51:47');
INSERT INTO `blog`.`authors` (`id`, `first_name`, `last_name`, `email`, `birthdate`, `added`) VALUES (2, 'Jamar', 'Buckridge', 'lebsack.noemie@example.net', '2016-04-25', '2017-06-11 04:40:50');
INSERT INTO `blog`.`authors` (`id`, `first_name`, `last_name`, `email`, `birthdate`, `added`) VALUES (3, 'Alivia', 'McLaughlin', 'landen.weber@example.com', '2010-01-21', '1980-01-31 06:20:19');
INSERT INTO `blog`.`authors` (`id`, `first_name`, `last_name`, `email`, `birthdate`, `added`) VALUES (4, 'Kristina', 'Schowalter', 'yhintz@example.com', '2005-12-25', '2010-12-14 11:03:54');
INSERT INTO `blog`.`authors` (`id`, `first_name`, `last_name`, `email`, `birthdate`, `added`) VALUES (5, 'Norris', 'Gleichner', 'derrick95@example.org', '1978-07-31', '2015-08-17 07:13:13');