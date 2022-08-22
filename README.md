# Will-Shark
> This is a lightweight go development framework

### How to start
> Configure items according to env settings
> 
> the Set directory is in `./envconfig`, for example setting the `dev_config.yaml`
```yaml
dev:
  mysql:
    will:
      Host: 172.16.161.54
      Port: 3306
      User: root
      Password: '123456'
      DataBase: will
      ParseTime: True
      MaxIdleConns: 10
      MaxOpenConns: 30
      ConnMaxLifetime: 28800
      ConnMaxIdletime: 7200
  redis:
    will:
      host: 172.16.161.54:6379
      password: ""
      database: 0
      maxIdleNum: 50
      maxActive: 5000
      maxIdleTimeout: 600
      connectTimeout: 1
      readTimeout: 2
  rocketmq:
    GroupName: test-rocket
    Topic: test-rocket
    Host:
      - 127.0.0.1:9876
    Retry: 3
````
> sql file
```sql
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for user
-- ----------------------------
DROP TABLE IF EXISTS `user`;
CREATE TABLE `user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL DEFAULT '',
  `age` int NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Records of user
-- ----------------------------
BEGIN;
INSERT INTO `user` (`id`, `name`, `age`) VALUES (1, 'will', 18);
COMMIT;

SET FOREIGN_KEY_CHECKS = 1;
````
> the run it
````
/private/var/folders/j9/plv1p__96fg_pf5fwx1f87200000gn/T/GoLand/___go_build_will
{"trace":{"welcome will's gang":"start the service with http in dev environment"},"appId":"100001","env":"dev","logType":"info"}
````
### What can it do
- mysql pool
- redis pool
- rocketMQ pool
- cron job
- wire

### What can it do the future
- grpc
- Prometheus
