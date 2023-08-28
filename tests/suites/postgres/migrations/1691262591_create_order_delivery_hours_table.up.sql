CREATE SEQUENCE IF NOT EXISTS order_delivery_hours_id_seq start 1 increment 1;

CREATE TABLE IF NOT EXISTS public.order_delivery_hours
(
    id bigint NOT NULL DEFAULT nextval('order_delivery_hours_id_seq'::regclass),
    order_id bigint,
    start_time time without time zone,
    end_time time without time zone,
    CONSTRAINT order_delivery_hours_pkey PRIMARY KEY (id),
    CONSTRAINT fk_orders_delivery_hours FOREIGN KEY (order_id)
        REFERENCES public.orders (id) MATCH SIMPLE
        ON UPDATE CASCADE
        ON DELETE CASCADE
)

TABLESPACE pg_default;