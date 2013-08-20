-- MySQL dump 10.14  Distrib 5.5.31-MariaDB, for Linux (x86_64)
--
-- Host: localhost    Database: bugzilla
-- ------------------------------------------------------
-- Server version	5.5.31-MariaDB

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `bugs`
--

DROP TABLE IF EXISTS `bugs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `bugs` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `status` varchar(10) DEFAULT 'new',
  `version` varchar(10) DEFAULT NULL,
  `severity` varchar(45) DEFAULT 'medium',
  `hardware` varchar(45) DEFAULT NULL,
  `priority` varchar(45) DEFAULT 'medium',
  `reporter` int(11) NOT NULL,
  `qa` int(11) DEFAULT NULL,
  `docs` int(11) DEFAULT NULL,
  `whiteboard` varchar(455) DEFAULT NULL,
  `summary` varchar(450) NOT NULL,
  `description` longtext NOT NULL,
  `reported` datetime NOT NULL,
  `fixedinver` varchar(255) DEFAULT NULL,
  `component_id` int(11) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_bugs_reporter_idx` (`reporter`),
  KEY `fk_bugs_qa_idx` (`qa`),
  KEY `fk_bugs_docs_idx` (`docs`),
  KEY `fk_bugs_component_id_idx` (`component_id`),
  CONSTRAINT `fk_bugs_component_id` FOREIGN KEY (`component_id`) REFERENCES `components` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_bugs_docs` FOREIGN KEY (`docs`) REFERENCES `users` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_bugs_qa` FOREIGN KEY (`qa`) REFERENCES `users` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_bugs_reporter` FOREIGN KEY (`reporter`) REFERENCES `users` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB AUTO_INCREMENT=616731 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `cc`
--

DROP TABLE IF EXISTS `cc`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `cc` (
  `bug_id` int(11) NOT NULL,
  `who` int(11) NOT NULL,
  PRIMARY KEY (`bug_id`,`who`),
  KEY `fk_cc_1_idx` (`bug_id`),
  KEY `fk_cc_2_idx` (`who`),
  CONSTRAINT `fk_cc_1` FOREIGN KEY (`bug_id`) REFERENCES `bugs` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_cc_2` FOREIGN KEY (`who`) REFERENCES `users` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `comments`
--

DROP TABLE IF EXISTS `comments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `comments` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `description` longtext NOT NULL,
  `user` int(11) NOT NULL,
  `datec` datetime NOT NULL,
  `bug` int(11) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_comments_user_idx` (`user`),
  KEY `fk_comments_bug_idx` (`bug`),
  CONSTRAINT `fk_comments_bug` FOREIGN KEY (`bug`) REFERENCES `bugs` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_comments_user` FOREIGN KEY (`user`) REFERENCES `users` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB AUTO_INCREMENT=117893 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `component_cc`
--

DROP TABLE IF EXISTS `component_cc`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `component_cc` (
  `component_id` int(11) NOT NULL,
  `user_id` int(11) NOT NULL,
  PRIMARY KEY (`component_id`,`user_id`),
  KEY `fk_component_cc_component_idx` (`component_id`),
  KEY `fk_component_cc_user_idx` (`user_id`),
  CONSTRAINT `fk_component_cc_component` FOREIGN KEY (`component_id`) REFERENCES `components` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_component_cc_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `components`
--

DROP TABLE IF EXISTS `components`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `components` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(45) CHARACTER SET latin1 DEFAULT NULL,
  `description` varchar(255) CHARACTER SET latin1 DEFAULT NULL,
  `product_id` int(11) DEFAULT NULL,
  `owner` int(11) NOT NULL,
  `qa` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_components_product_idx` (`product_id`),
  KEY `fk_components_1_idx` (`owner`),
  KEY `fk_components_2_idx` (`qa`),
  CONSTRAINT `fk_components_1` FOREIGN KEY (`owner`) REFERENCES `users` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_components_2` FOREIGN KEY (`qa`) REFERENCES `users` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_components_product` FOREIGN KEY (`product_id`) REFERENCES `products` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB AUTO_INCREMENT=16078 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dependencies`
--

DROP TABLE IF EXISTS `dependencies`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dependencies` (
  `blocked` int(11) NOT NULL,
  `dependson` int(11) NOT NULL,
  PRIMARY KEY (`blocked`),
  KEY `fk_dependencies_dep_idx` (`dependson`),
  CONSTRAINT `fk_dependencies_dep` FOREIGN KEY (`dependson`) REFERENCES `bugs` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_dependencies_block` FOREIGN KEY (`blocked`) REFERENCES `bugs` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `duplicates`
--

DROP TABLE IF EXISTS `duplicates`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `duplicates` (
  `dup` int(11) NOT NULL,
  `dup_of` int(11) NOT NULL,
  PRIMARY KEY (`dup`),
  KEY `fk_duplicates_dup_of_idx` (`dup_of`),
  CONSTRAINT `fk_duplicates_dup` FOREIGN KEY (`dup`) REFERENCES `bugs` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_duplicates_dup_of` FOREIGN KEY (`dup_of`) REFERENCES `bugs` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `products`
--

DROP TABLE IF EXISTS `products`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `products` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) DEFAULT NULL,
  `description` varchar(500) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `users` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(64) CHARACTER SET latin1 DEFAULT NULL,
  `email` varchar(255) CHARACTER SET latin1 NOT NULL,
  `type` varchar(19) DEFAULT '1',
  `password` varchar(255) CHARACTER SET latin1 NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2013-08-20 10:16:38
