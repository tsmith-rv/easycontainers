CREATE DATABASE `blog`;

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

CREATE TABLE `blog`.`posts` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `author_id` int(11) NOT NULL,
  `title` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `description` varchar(500) COLLATE utf8_unicode_ci NOT NULL,
  `content` text COLLATE utf8_unicode_ci NOT NULL,
  `date` varchar(50) COLLATE utf8_unicode_ci NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=101 DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;

INSERT INTO `blog`.`posts` (`id`, `author_id`, `title`, `description`, `content`, `date`) VALUES (1, 1, 'Cupiditate ducimus magni error aspernatur quam eaque officia recusandae.', 'Eligendi quo harum in laboriosam voluptatum ut nemo ex. Et sapiente magni praesentium libero. Et sunt et veritatis unde quos perspiciatis amet ut.', 'Asperiores rerum harum laborum at qui quae quia. Iusto aliquam sapiente nesciunt laboriosam expedita. Eos qui delectus dolorum eligendi ipsam ad.', '1975-07-21');
INSERT INTO `blog`.`posts` (`id`, `author_id`, `title`, `description`, `content`, `date`) VALUES (2, 2, 'Dignissimos eius voluptatem aliquid ab nostrum facere saepe voluptatem.', 'Dolorem aut et inventore rem. Ut eius eveniet qui. Error velit ea corrupti voluptas laboriosam aliquam. Blanditiis aliquam voluptas consequatur quas voluptatem.', 'Delectus qui non nesciunt ut sit omnis a. Mollitia iste ullam illum ipsam. At et voluptatibus dolores repudiandae officiis.', '1996-01-10');
INSERT INTO `blog`.`posts` (`id`, `author_id`, `title`, `description`, `content`, `date`) VALUES (3, 3, 'Voluptas modi consequatur est id.', 'Sit culpa nemo repudiandae sint minus id. Velit eveniet aliquam tempora modi. Laboriosam molestiae ut aut omnis.', 'Qui et est recusandae qui ut in nesciunt. Maxime dolorem eligendi consectetur est dicta excepturi. Incidunt ut vel necessitatibus.', '1996-03-21');