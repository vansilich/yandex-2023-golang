CREATE SEQUENCE IF NOT EXISTS orders_id_seq start 1 increment 1;

CREATE TABLE IF NOT EXISTS public.orders
(
    id bigint NOT NULL DEFAULT nextval('orders_id_seq'::regclass),
    weight numeric NOT NULL,
    regions integer NOT NULL,
    cost bigint NOT NULL,
    completed_time timestamp with time zone,
    delivery_group_id bigint,
    CONSTRAINT orders_pkey PRIMARY KEY (id),
    CONSTRAINT fk_orders_delivery_group FOREIGN KEY (delivery_group_id)
        REFERENCES public.delivery_groups (id) MATCH SIMPLE
        ON UPDATE CASCADE
        ON DELETE CASCADE
)

TABLESPACE pg_default;