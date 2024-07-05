CREATE TABLE gauge (
    metric_id SERIAL PRIMARY KEY,
    metric_name VARCHAR NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL
);

CREATE TABLE counter (
   metric_id SERIAL PRIMARY KEY,
   metric_name VARCHAR NOT NULL,
   metric_value BIGINT NOT NULL
);