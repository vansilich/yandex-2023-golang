CREATE SEQUENCE IF NOT EXISTS courier_working_hours_id_seq start 1 increment 1;

CREATE TABLE IF NOT EXISTS public.courier_working_hours
(
    id bigint NOT NULL DEFAULT nextval('courier_working_hours_id_seq'::regclass),
    courier_id bigint,
    start_time time without time zone,
    end_time time without time zone,
    CONSTRAINT courier_working_hours_pkey PRIMARY KEY (id),
    CONSTRAINT fk_couriers_working_hours FOREIGN KEY (courier_id)
        REFERENCES public.couriers (id) MATCH SIMPLE
        ON UPDATE CASCADE
        ON DELETE CASCADE
)

TABLESPACE pg_default;