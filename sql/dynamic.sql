CREATE TABLE `task` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `args` text NOT NULL,
  `cmd` varchar(255) NOT NULL DEFAULT '',
  `errfile` varchar(255) NOT NULL DEFAULT '',
  `outfile` varchar(255) NOT NULL DEFAULT '',
  `type` varchar(255) NOT NULL DEFAULT '',
  `do_once` tinyint(1) NOT NULL DEFAULT '1' COMMENT '1 执行多次 2 执行一次',
  `pid` int(10) NOT NULL DEFAULT '0',
  `job_id` varchar(100) NOT NULL DEFAULT '',
  `dc` varchar(255) NOT NULL DEFAULT '',
  `node` varchar(255) NOT NULL DEFAULT '',
  `ip` varchar(255) NOT NULL DEFAULT '',
  `load_method` varchar(255) DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  `update_time` datetime DEFAULT NULL,
  `heart_beat_time` datetime DEFAULT NULL,
  `big_one` varchar(255) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `job_index` (`job_id`,`pid`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1022 DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ann_log_statistics` (
                                    `id` int(8) NOT NULL AUTO_INCREMENT,
                                    `tenantId` varchar(64) DEFAULT NULL,
                                    `circuitId` varchar(64) DEFAULT NULL,
                                    `serviceEnable` varchar(64) DEFAULT NULL,
                                    `bandwidth_in` bigint(64) DEFAULT NULL,
                                    `bandwidth_out` bigint(64) DEFAULT NULL,
                                    `traffic_in` bigint(64) DEFAULT NULL,
                                    `traffic_out` bigint(64) DEFAULT NULL,
                                    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;


CREATE TABLE `ann_order` (
                           `id` int(8) NOT NULL AUTO_INCREMENT,
                           `tenantId` varchar(64) DEFAULT NULL,
                           `circuitId` varchar(64) DEFAULT NULL,
                           `orderNumber` varchar(64) DEFAULT NULL,
                           `orderType` varchar(64) DEFAULT NULL,
                           `orderConfigType` varchar(64) DEFAULT NULL,
                           `trafficSize` varchar(64) DEFAULT NULL,
                           `bandwidth` varchar(64) DEFAULT NULL,
                           `contractType` int(64) DEFAULT NULL,
                           `contractPeriod` int(64) DEFAULT NULL,
                           `contractPeriodUnit` varchar(64) DEFAULT NULL,
                           `autoRenew` int(2) DEFAULT NULL,
                           PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;


CREATE TABLE `ann_order_config` (
                                  `id` int(8) NOT NULL AUTO_INCREMENT,
                                  `tenantId` varchar(64) DEFAULT NULL,
                                  `circuitId` varchar(64) DEFAULT NULL,
                                  `serviceEnable` varchar(64) DEFAULT NULL,
                                  `orderType` varchar(64) DEFAULT NULL,
                                  `cnameScheduleEnable` varchar(64) DEFAULT NULL,
                                  `requestDomain` varchar(64) DEFAULT NULL,
                                  `wildcardDomainEnable` varchar(64) DEFAULT NULL,
                                  `serviceMode` varchar(64) DEFAULT NULL,
                                  `servers` varchar(64) DEFAULT NULL,
                                  `standbyServers` varchar(64) DEFAULT NULL,
                                  `layerType` varchar(64) DEFAULT NULL,
                                  `accessPort` varchar(64) DEFAULT NULL,
                                  `cpeId` varchar(128) DEFAULT NULL,
                                  `cpeStatus` varchar(64) DEFAULT NULL,
                                  `demandTenantId` varchar(128) DEFAULT NULL,
                                  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT;