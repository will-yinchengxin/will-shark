CREATE TABLE ws_gen_crud_probe (
    code varchar(24) NOT NULL PRIMARY KEY,
    title varchar(50) NOT NULL,
    status int NOT NULL,
    balance decimal(10,2) NOT NULL,
    attempts int NOT NULL DEFAULT 18,
    enabled boolean NOT NULL
);
CREATE TABLE ws_gen_int_probe (
    id bigint NOT NULL PRIMARY KEY,
    title varchar(50) NOT NULL
);
