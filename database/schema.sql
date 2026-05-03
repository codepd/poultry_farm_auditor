--
-- PostgreSQL database dump
--

\restrict 1NJsD98LqX0tmGnDc92U3HS994Wcywm3rRU4hDb6YmsKbzCyQtzrfXcNk72ht04

-- Dumped from database version 15.15
-- Dumped by pg_dump version 15.15

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


--
-- Name: category_enum; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.category_enum AS ENUM (
    'EGG',
    'FEED',
    'MEDICINE',
    'OTHER'
);


--
-- Name: price_type_enum; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.price_type_enum AS ENUM (
    'EGG',
    'FEED'
);


--
-- Name: transaction_type_enum; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.transaction_type_enum AS ENUM (
    'SALE',
    'PURCHASE',
    'PAYMENT',
    'TDS',
    'DISCOUNT',
    'EXPENSE',
    'INCOME'
);


--
-- Name: user_role_enum; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.user_role_enum AS ENUM (
    'ADMIN',
    'OWNER',
    'CO_OWNER',
    'OTHER_USER',
    'AUDITOR',
    'MANAGER'
);


--
-- Name: update_batch_count_on_mortality(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_batch_count_on_mortality() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE hen_batches
    SET current_count = current_count - NEW.count,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.batch_id;
    RETURN NEW;
END;
$$;


--
-- Name: update_hen_batches_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_hen_batches_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;


--
-- Name: update_ledger_parses_from_transactions(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_ledger_parses_from_transactions() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    t_year INTEGER;
    t_month INTEGER;
    t_tenant_id UUID;
    lp_id INTEGER;
BEGIN
    -- Get year, month, tenant_id from the transaction
    IF TG_OP = 'DELETE' THEN
        t_year := EXTRACT(YEAR FROM OLD.transaction_date);
        t_month := EXTRACT(MONTH FROM OLD.transaction_date);
        t_tenant_id := OLD.tenant_id;
    ELSE
        t_year := EXTRACT(YEAR FROM NEW.transaction_date);
        t_month := EXTRACT(MONTH FROM NEW.transaction_date);
        t_tenant_id := NEW.tenant_id;
    END IF;
    
    -- Find the ledger_parse record for this month/year/tenant
    SELECT id INTO lp_id
    FROM ledger_parses
    WHERE tenant_id = t_tenant_id
        AND year = t_year
        AND month = t_month
    LIMIT 1;
    
    -- If ledger_parse exists, recalculate totals
    IF lp_id IS NOT NULL THEN
        -- Recalculate totals from transactions
        UPDATE ledger_parses SET
            total_eggs = (
                SELECT COALESCE(SUM(amount), 0)
                FROM transactions
                WHERE tenant_id = t_tenant_id
                    AND category = 'EGG'
                    AND transaction_type = 'SALE'
                    AND EXTRACT(YEAR FROM transaction_date) = t_year
                    AND EXTRACT(MONTH FROM transaction_date) = t_month
            ),
            total_feeds = (
                SELECT COALESCE(SUM(amount), 0)
                FROM transactions
                WHERE tenant_id = t_tenant_id
                    AND category = 'FEED'
                    AND transaction_type = 'PURCHASE'
                    AND EXTRACT(YEAR FROM transaction_date) = t_year
                    AND EXTRACT(MONTH FROM transaction_date) = t_month
            ),
            total_medicines = (
                SELECT COALESCE(SUM(amount), 0)
                FROM transactions
                WHERE tenant_id = t_tenant_id
                    AND category = 'MEDICINE'
                    AND EXTRACT(YEAR FROM transaction_date) = t_year
                    AND EXTRACT(MONTH FROM transaction_date) = t_month
            ),
            net_profit = (
                SELECT COALESCE(SUM(CASE WHEN category = 'EGG' AND transaction_type = 'SALE' THEN amount ELSE 0 END), 0) +
                       COALESCE(SUM(CASE WHEN transaction_type = 'DISCOUNT' THEN amount ELSE 0 END), 0) -
                       COALESCE(SUM(CASE WHEN category = 'FEED' AND transaction_type = 'PURCHASE' THEN amount ELSE 0 END), 0) -
                       COALESCE(SUM(CASE WHEN category = 'MEDICINE' THEN amount ELSE 0 END), 0) -
                       COALESCE(SUM(CASE WHEN category = 'OTHER' AND transaction_type NOT IN ('PAYMENT', 'TDS', 'DISCOUNT') THEN amount ELSE 0 END), 0) -
                       COALESCE(SUM(CASE WHEN transaction_type = 'TDS' THEN amount ELSE 0 END), 0)
                FROM transactions
                WHERE tenant_id = t_tenant_id
                    AND EXTRACT(YEAR FROM transaction_date) = t_year
                    AND EXTRACT(MONTH FROM transaction_date) = t_month
            )
        WHERE id = lp_id;
        
        -- Update breakdowns
        -- Delete existing breakdowns for this ledger
        DELETE FROM ledger_breakdowns WHERE ledger_parse_id = lp_id;
        
        -- Insert egg breakdowns
        INSERT INTO ledger_breakdowns (ledger_parse_id, breakdown_type, quantity)
        SELECT lp_id, 'EGG_LARGE', COALESCE(SUM(quantity), 0)
        FROM transactions
        WHERE tenant_id = t_tenant_id
            AND category = 'EGG'
            AND transaction_type = 'SALE'
            AND (item_name LIKE '%LARGE EGG%' OR item_name LIKE '%CORRECT%' OR item_name LIKE '%EXPORT%')
            AND EXTRACT(YEAR FROM transaction_date) = t_year
            AND EXTRACT(MONTH FROM transaction_date) = t_month
        HAVING COALESCE(SUM(quantity), 0) > 0;
        
        INSERT INTO ledger_breakdowns (ledger_parse_id, breakdown_type, quantity)
        SELECT lp_id, 'EGG_MEDIUM', COALESCE(SUM(quantity), 0)
        FROM transactions
        WHERE tenant_id = t_tenant_id
            AND category = 'EGG'
            AND transaction_type = 'SALE'
            AND item_name LIKE '%MEDIUM EGG%'
            AND EXTRACT(YEAR FROM transaction_date) = t_year
            AND EXTRACT(MONTH FROM transaction_date) = t_month
        HAVING COALESCE(SUM(quantity), 0) > 0;
        
        INSERT INTO ledger_breakdowns (ledger_parse_id, breakdown_type, quantity)
        SELECT lp_id, 'EGG_SMALL', COALESCE(SUM(quantity), 0)
        FROM transactions
        WHERE tenant_id = t_tenant_id
            AND category = 'EGG'
            AND transaction_type = 'SALE'
            AND item_name LIKE '%SMALL EGG%'
            AND EXTRACT(YEAR FROM transaction_date) = t_year
            AND EXTRACT(MONTH FROM transaction_date) = t_month
        HAVING COALESCE(SUM(quantity), 0) > 0;
        
        -- Insert feed breakdowns by type
        INSERT INTO ledger_breakdowns (ledger_parse_id, breakdown_type, quantity)
        SELECT lp_id, 'FEED_PRE_LAYER_MASH', COALESCE(SUM(quantity), 0)
        FROM transactions
        WHERE tenant_id = t_tenant_id
            AND category = 'FEED'
            AND transaction_type = 'PURCHASE'
            AND (item_name LIKE '%PRE%LAYER%' OR item_name LIKE '%PRE-LAYER%')
            AND (unit LIKE '%KG%' OR unit IS NULL)
            AND EXTRACT(YEAR FROM transaction_date) = t_year
            AND EXTRACT(MONTH FROM transaction_date) = t_month
        HAVING COALESCE(SUM(quantity), 0) > 0;
        
        INSERT INTO ledger_breakdowns (ledger_parse_id, breakdown_type, quantity)
        SELECT lp_id, 'FEED_LAYER_MASH', COALESCE(SUM(quantity), 0)
        FROM transactions
        WHERE tenant_id = t_tenant_id
            AND category = 'FEED'
            AND transaction_type = 'PURCHASE'
            AND item_name LIKE '%LAYER%'
            AND item_name NOT LIKE '%PRE%'
            AND (unit LIKE '%KG%' OR unit IS NULL)
            AND EXTRACT(YEAR FROM transaction_date) = t_year
            AND EXTRACT(MONTH FROM transaction_date) = t_month
        HAVING COALESCE(SUM(quantity), 0) > 0;
        
        INSERT INTO ledger_breakdowns (ledger_parse_id, breakdown_type, quantity)
        SELECT lp_id, 'FEED_GROWER_MASH', COALESCE(SUM(quantity), 0)
        FROM transactions
        WHERE tenant_id = t_tenant_id
            AND category = 'FEED'
            AND transaction_type = 'PURCHASE'
            AND item_name LIKE '%GROWER%'
            AND (unit LIKE '%KG%' OR unit IS NULL)
            AND EXTRACT(YEAR FROM transaction_date) = t_year
            AND EXTRACT(MONTH FROM transaction_date) = t_month
        HAVING COALESCE(SUM(quantity), 0) > 0;
        
        INSERT INTO ledger_breakdowns (ledger_parse_id, breakdown_type, quantity)
        SELECT lp_id, 'FEED_CHICK_MASH', COALESCE(SUM(quantity), 0)
        FROM transactions
        WHERE tenant_id = t_tenant_id
            AND category = 'FEED'
            AND transaction_type = 'PURCHASE'
            AND item_name LIKE '%CHICK%'
            AND (unit LIKE '%KG%' OR unit IS NULL)
            AND EXTRACT(YEAR FROM transaction_date) = t_year
            AND EXTRACT(MONTH FROM transaction_date) = t_month
        HAVING COALESCE(SUM(quantity), 0) > 0;
    END IF;
    
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: hen_batches; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.hen_batches (
    id integer NOT NULL,
    tenant_id uuid NOT NULL,
    batch_name character varying(255) NOT NULL,
    initial_count integer NOT NULL,
    current_count integer NOT NULL,
    age_weeks integer DEFAULT 0,
    age_days integer DEFAULT 0,
    date_added date NOT NULL,
    notes text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: hen_batches_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.hen_batches_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: hen_batches_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.hen_batches_id_seq OWNED BY public.hen_batches.id;


--
-- Name: hen_mortality; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.hen_mortality (
    id integer NOT NULL,
    batch_id integer NOT NULL,
    mortality_date date NOT NULL,
    count integer NOT NULL,
    reason character varying(255),
    notes text,
    recorded_by_user_id integer,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: hen_mortality_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.hen_mortality_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: hen_mortality_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.hen_mortality_id_seq OWNED BY public.hen_mortality.id;


--
-- Name: ledger_breakdowns; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ledger_breakdowns (
    id integer NOT NULL,
    ledger_parse_id integer NOT NULL,
    breakdown_type character varying(50) NOT NULL,
    quantity numeric(12,3) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: ledger_breakdowns_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.ledger_breakdowns_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: ledger_breakdowns_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.ledger_breakdowns_id_seq OWNED BY public.ledger_breakdowns.id;


--
-- Name: ledger_parses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ledger_parses (
    id integer NOT NULL,
    pdf_filename character varying(255) NOT NULL,
    parse_date date NOT NULL,
    month integer NOT NULL,
    year integer NOT NULL,
    opening_balance numeric(12,2),
    closing_balance numeric(12,2),
    total_eggs numeric(12,2),
    total_feeds numeric(12,2),
    total_medicines numeric(12,2),
    net_profit numeric(12,2),
    parsed_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    eggs_large_qty numeric(12,3),
    eggs_medium_qty numeric(12,3),
    eggs_small_qty numeric(12,3),
    feeds_total_kg numeric(12,3),
    tenant_id uuid,
    CONSTRAINT ledger_parses_month_check CHECK (((month >= 1) AND (month <= 12)))
);


--
-- Name: ledger_parses_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.ledger_parses_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: ledger_parses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.ledger_parses_id_seq OWNED BY public.ledger_parses.id;


--
-- Name: price_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.price_history (
    id integer NOT NULL,
    price_date date NOT NULL,
    price_type public.price_type_enum NOT NULL,
    item_name character varying(255) NOT NULL,
    price numeric(12,2) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid
);


--
-- Name: price_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.price_history_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: price_history_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.price_history_id_seq OWNED BY public.price_history.id;


--
-- Name: role_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role_permissions (
    id integer NOT NULL,
    tenant_id uuid NOT NULL,
    role public.user_role_enum NOT NULL,
    can_view_sensitive_data boolean DEFAULT false,
    can_edit_transactions boolean DEFAULT false,
    can_approve_transactions boolean DEFAULT false,
    can_manage_users boolean DEFAULT false,
    can_view_charts boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: role_permissions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.role_permissions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: role_permissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.role_permissions_id_seq OWNED BY public.role_permissions.id;


--
-- Name: tenant_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenant_items (
    id integer NOT NULL,
    tenant_id uuid NOT NULL,
    category public.category_enum NOT NULL,
    item_name character varying(255) NOT NULL,
    display_order integer DEFAULT 0,
    is_active boolean DEFAULT true,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: tenant_items_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.tenant_items_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: tenant_items_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.tenant_items_id_seq OWNED BY public.tenant_items.id;


--
-- Name: tenant_users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenant_users (
    id integer NOT NULL,
    tenant_id uuid NOT NULL,
    user_id integer NOT NULL,
    role public.user_role_enum NOT NULL,
    is_owner boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: tenant_users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.tenant_users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: tenant_users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.tenant_users_id_seq OWNED BY public.tenant_users.id;


--
-- Name: tenant_uuid; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenant_uuid (
    id uuid
);


--
-- Name: tenants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenants (
    name character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    id uuid NOT NULL,
    timezone character varying(50) DEFAULT 'Asia/Kolkata'::character varying,
    egg_price_reference_zone character varying(100) DEFAULT 'Namakkal'::character varying,
    financial_year_start_month integer DEFAULT 4,
    age_category_chick_max_weeks integer DEFAULT 6,
    age_category_grower_max_weeks integer DEFAULT 18,
    age_category_prelayer_max_weeks integer DEFAULT 22,
    CONSTRAINT tenants_financial_year_start_month_check CHECK (((financial_year_start_month >= 1) AND (financial_year_start_month <= 12)))
);


--
-- Name: COLUMN tenants.timezone; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tenants.timezone IS 'IANA timezone identifier (e.g., Asia/Kolkata, America/New_York)';
COMMENT ON COLUMN public.tenants.egg_price_reference_zone IS 'Preferred NECC zone for tenant egg price import (e.g., Namakkal, Hyderabad)';
COMMENT ON COLUMN public.tenants.financial_year_start_month IS 'Financial year start month (1-12). Example: 4 for April-March financial year.';


--
-- Name: transactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.transactions (
    id integer NOT NULL,
    transaction_date date NOT NULL,
    transaction_type public.transaction_type_enum NOT NULL,
    category public.category_enum NOT NULL,
    item_name character varying(255) NOT NULL,
    quantity numeric(12,3),
    unit character varying(50),
    rate numeric(12,2),
    amount numeric(12,2) NOT NULL,
    notes text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid,
    payment_date date,
    period_month date,
    period_week integer,
    period_days integer
);


--
-- Name: COLUMN transactions.payment_date; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.transactions.payment_date IS 'Date when payment was made. Defaults to transaction_date if not specified.';


--
-- Name: COLUMN transactions.period_month; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.transactions.period_month IS 'Month the payment is for (stored as first day of month). Defaults to payment_date month if not specified.';


--
-- Name: COLUMN transactions.period_week; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.transactions.period_week IS 'Week number within the payment period (optional).';


--
-- Name: COLUMN transactions.period_days; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.transactions.period_days IS 'Number of days the payment covers (optional).';


--
-- Name: transactions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.transactions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: transactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.transactions_id_seq OWNED BY public.transactions.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id integer NOT NULL,
    email character varying(255) NOT NULL,
    password_hash character varying(255),
    full_name character varying(255),
    is_active boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: hen_batches id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hen_batches ALTER COLUMN id SET DEFAULT nextval('public.hen_batches_id_seq'::regclass);


--
-- Name: hen_mortality id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hen_mortality ALTER COLUMN id SET DEFAULT nextval('public.hen_mortality_id_seq'::regclass);


--
-- Name: ledger_breakdowns id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ledger_breakdowns ALTER COLUMN id SET DEFAULT nextval('public.ledger_breakdowns_id_seq'::regclass);


--
-- Name: ledger_parses id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ledger_parses ALTER COLUMN id SET DEFAULT nextval('public.ledger_parses_id_seq'::regclass);


--
-- Name: price_history id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.price_history ALTER COLUMN id SET DEFAULT nextval('public.price_history_id_seq'::regclass);


--
-- Name: role_permissions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions ALTER COLUMN id SET DEFAULT nextval('public.role_permissions_id_seq'::regclass);


--
-- Name: tenant_items id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_items ALTER COLUMN id SET DEFAULT nextval('public.tenant_items_id_seq'::regclass);


--
-- Name: tenant_users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_users ALTER COLUMN id SET DEFAULT nextval('public.tenant_users_id_seq'::regclass);


--
-- Name: transactions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions ALTER COLUMN id SET DEFAULT nextval('public.transactions_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: hen_batches hen_batches_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hen_batches
    ADD CONSTRAINT hen_batches_pkey PRIMARY KEY (id);


--
-- Name: hen_mortality hen_mortality_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hen_mortality
    ADD CONSTRAINT hen_mortality_pkey PRIMARY KEY (id);


--
-- Name: ledger_breakdowns ledger_breakdowns_ledger_parse_id_breakdown_type_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ledger_breakdowns
    ADD CONSTRAINT ledger_breakdowns_ledger_parse_id_breakdown_type_key UNIQUE (ledger_parse_id, breakdown_type);


--
-- Name: ledger_breakdowns ledger_breakdowns_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ledger_breakdowns
    ADD CONSTRAINT ledger_breakdowns_pkey PRIMARY KEY (id);


--
-- Name: ledger_parses ledger_parses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ledger_parses
    ADD CONSTRAINT ledger_parses_pkey PRIMARY KEY (id);


--
-- Name: price_history price_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.price_history
    ADD CONSTRAINT price_history_pkey PRIMARY KEY (id);


--
-- Name: price_history price_history_tenant_id_price_date_price_type_item_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.price_history
    ADD CONSTRAINT price_history_tenant_id_price_date_price_type_item_name_key UNIQUE (tenant_id, price_date, price_type, item_name);


--
-- Name: role_permissions role_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_pkey PRIMARY KEY (id);


--
-- Name: role_permissions role_permissions_tenant_id_role_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_tenant_id_role_key UNIQUE (tenant_id, role);


--
-- Name: tenant_items tenant_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_items
    ADD CONSTRAINT tenant_items_pkey PRIMARY KEY (id);


--
-- Name: tenant_items tenant_items_tenant_id_category_item_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_items
    ADD CONSTRAINT tenant_items_tenant_id_category_item_name_key UNIQUE (tenant_id, category, item_name);


--
-- Name: tenant_users tenant_users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_users
    ADD CONSTRAINT tenant_users_pkey PRIMARY KEY (id);


--
-- Name: tenant_users tenant_users_tenant_id_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_users
    ADD CONSTRAINT tenant_users_tenant_id_user_id_key UNIQUE (tenant_id, user_id);


--
-- Name: tenants tenants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_pkey PRIMARY KEY (id);


--
-- Name: transactions transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_pkey PRIMARY KEY (id);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_hen_batches_date_added; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_hen_batches_date_added ON public.hen_batches USING btree (date_added);


--
-- Name: idx_hen_batches_tenant_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_hen_batches_tenant_id ON public.hen_batches USING btree (tenant_id);


--
-- Name: idx_hen_mortality_batch_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_hen_mortality_batch_id ON public.hen_mortality USING btree (batch_id);


--
-- Name: idx_tenant_items_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenant_items_active ON public.tenant_items USING btree (tenant_id, category, is_active) WHERE (is_active = true);


--
-- Name: idx_tenant_items_tenant_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tenant_items_tenant_category ON public.tenant_items USING btree (tenant_id, category);


--
-- Name: hen_mortality trigger_update_batch_count_on_mortality; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_update_batch_count_on_mortality AFTER INSERT ON public.hen_mortality FOR EACH ROW EXECUTE FUNCTION public.update_batch_count_on_mortality();


--
-- Name: hen_batches trigger_update_hen_batches_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_update_hen_batches_updated_at BEFORE UPDATE ON public.hen_batches FOR EACH ROW EXECUTE FUNCTION public.update_hen_batches_updated_at();


--
-- Name: transactions trigger_update_ledger_on_transaction_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_update_ledger_on_transaction_delete AFTER DELETE ON public.transactions FOR EACH ROW EXECUTE FUNCTION public.update_ledger_parses_from_transactions();


--
-- Name: transactions trigger_update_ledger_on_transaction_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_update_ledger_on_transaction_insert AFTER INSERT ON public.transactions FOR EACH ROW EXECUTE FUNCTION public.update_ledger_parses_from_transactions();


--
-- Name: transactions trigger_update_ledger_on_transaction_update; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_update_ledger_on_transaction_update AFTER UPDATE ON public.transactions FOR EACH ROW WHEN (((old.amount IS DISTINCT FROM new.amount) OR (old.quantity IS DISTINCT FROM new.quantity) OR (old.category IS DISTINCT FROM new.category) OR (old.transaction_type IS DISTINCT FROM new.transaction_type) OR (old.transaction_date IS DISTINCT FROM new.transaction_date))) EXECUTE FUNCTION public.update_ledger_parses_from_transactions();


--
-- Name: hen_batches hen_batches_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hen_batches
    ADD CONSTRAINT hen_batches_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: hen_mortality hen_mortality_batch_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hen_mortality
    ADD CONSTRAINT hen_mortality_batch_id_fkey FOREIGN KEY (batch_id) REFERENCES public.hen_batches(id) ON DELETE CASCADE;


--
-- Name: ledger_breakdowns ledger_breakdowns_ledger_parse_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ledger_breakdowns
    ADD CONSTRAINT ledger_breakdowns_ledger_parse_id_fkey FOREIGN KEY (ledger_parse_id) REFERENCES public.ledger_parses(id) ON DELETE CASCADE;


--
-- Name: role_permissions role_permissions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: tenant_items tenant_items_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_items
    ADD CONSTRAINT tenant_items_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: tenant_users tenant_users_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_users
    ADD CONSTRAINT tenant_users_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;


--
-- Name: tenant_users tenant_users_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenant_users
    ADD CONSTRAINT tenant_users_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: update_batch_count_on_sale(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE OR REPLACE FUNCTION public.update_batch_count_on_sale() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    remaining_count integer;
BEGIN
    IF NEW.count <= 0 THEN
        RAISE EXCEPTION 'Sale count must be greater than 0';
    END IF;

    SELECT current_count - NEW.count
    INTO remaining_count
    FROM hen_batches
    WHERE id = NEW.batch_id
    FOR UPDATE;

    IF remaining_count IS NULL THEN
        RAISE EXCEPTION 'Hen batch % not found', NEW.batch_id;
    END IF;

    IF remaining_count < 0 THEN
        RAISE EXCEPTION 'Cannot sell % hens from batch %, only % hens available',
            NEW.count, NEW.batch_id, remaining_count + NEW.count;
    END IF;

    UPDATE hen_batches
    SET current_count = remaining_count,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.batch_id;

    RETURN NEW;
END;
$$;


--
-- Name: hen_batch_sales; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE IF NOT EXISTS public.hen_batch_sales (
    id integer GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    batch_id integer NOT NULL,
    sale_date date NOT NULL,
    count integer NOT NULL CHECK (count > 0),
    price_per_hen numeric(12,2) NOT NULL CHECK (price_per_hen >= 0),
    total_amount numeric(12,2) NOT NULL CHECK (total_amount >= 0),
    notes text,
    recorded_by_user_id integer,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: idx_hen_batch_sales_batch_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_hen_batch_sales_batch_id ON public.hen_batch_sales USING btree (batch_id);


--
-- Name: hen_batch_sales trigger_update_batch_count_on_sale; Type: TRIGGER; Schema: public; Owner: -
--

DROP TRIGGER IF EXISTS trigger_update_batch_count_on_sale ON public.hen_batch_sales;
CREATE TRIGGER trigger_update_batch_count_on_sale
AFTER INSERT ON public.hen_batch_sales
FOR EACH ROW EXECUTE FUNCTION public.update_batch_count_on_sale();


--
-- Name: hen_batch_sales hen_batch_sales_batch_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'hen_batch_sales_batch_id_fkey'
    ) THEN
        ALTER TABLE ONLY public.hen_batch_sales
            ADD CONSTRAINT hen_batch_sales_batch_id_fkey
            FOREIGN KEY (batch_id) REFERENCES public.hen_batches(id) ON DELETE CASCADE;
    END IF;
END$$;


--
-- PostgreSQL database dump complete
--

\unrestrict 1NJsD98LqX0tmGnDc92U3HS994Wcywm3rRU4hDb6YmsKbzCyQtzrfXcNk72ht04

