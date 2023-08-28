CREATE SEQUENCE IF NOT EXISTS delivery_groups_id_seq start 1 increment 1;

CREATE TABLE IF NOT EXISTS public.delivery_groups
(
    id bigint NOT NULL DEFAULT nextval('delivery_groups_id_seq'::regclass),
    courier_id bigint NOT NULL,
    courier_working_hours_id bigint NOT NULL,
    assign_date date NOT NULL,
    start_date_time timestamp with time zone NOT NULL,
    end_date_time timestamp with time zone NOT NULL,
    CONSTRAINT delivery_groups_pkey PRIMARY KEY (id),
    CONSTRAINT fk_delivery_groups_courier FOREIGN KEY (courier_id)
        REFERENCES public.couriers (id) MATCH SIMPLE
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    CONSTRAINT fk_delivery_groups_working_hours FOREIGN KEY (courier_working_hours_id)
        REFERENCES public.courier_working_hours (id) MATCH SIMPLE
        ON UPDATE CASCADE
        ON DELETE CASCADE
)

TABLESPACE pg_default;