CREATE SEQUENCE IF NOT EXISTS couriers_id_seq start 1 increment 1;

CREATE TABLE IF NOT EXISTS public.couriers
(
    id bigint NOT NULL DEFAULT nextval('couriers_id_seq'::regclass),
    courier_type text COLLATE pg_catalog."default",
    regions integer[],
    CONSTRAINT couriers_pkey PRIMARY KEY (id)
)

TABLESPACE pg_default;