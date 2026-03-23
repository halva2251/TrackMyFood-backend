-- Food Flight Tracker schema

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Producers
CREATE TABLE producers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    location   VARCHAR(255),
    country    VARCHAR(2)   NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Products
CREATE TABLE products (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    producer_id           UUID         NOT NULL REFERENCES producers(id),
    name                  VARCHAR(255) NOT NULL,
    category              VARCHAR(100) NOT NULL,
    barcode               VARCHAR(20)  NOT NULL UNIQUE,
    min_temp_celsius      DECIMAL(4,1),
    max_temp_celsius      DECIMAL(4,1),
    optimal_shelf_hours   INTEGER,
    optimal_handling_steps INTEGER,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_producer_id ON products(producer_id);

-- Batches
CREATE TABLE batches (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id              UUID         NOT NULL REFERENCES products(id),
    lot_number              VARCHAR(50)  NOT NULL,
    production_date         TIMESTAMPTZ  NOT NULL,
    expiry_date             TIMESTAMPTZ,
    trust_score             DECIMAL(5,2) CHECK (trust_score BETWEEN 0 AND 100),
    sub_score_cold_chain    DECIMAL(5,2) CHECK (sub_score_cold_chain BETWEEN 0 AND 100),
    sub_score_quality       DECIMAL(5,2) CHECK (sub_score_quality BETWEEN 0 AND 100),
    sub_score_time_to_shelf DECIMAL(5,2) CHECK (sub_score_time_to_shelf BETWEEN 0 AND 100),
    sub_score_producer      DECIMAL(5,2) CHECK (sub_score_producer BETWEEN 0 AND 100),
    sub_score_handling      DECIMAL(5,2) CHECK (sub_score_handling BETWEEN 0 AND 100),
    score_calculated_at     TIMESTAMPTZ,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, lot_number)
);

CREATE INDEX idx_batches_product_id ON batches(product_id);

-- Journey steps
CREATE TABLE journey_steps (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id    UUID         NOT NULL REFERENCES batches(id),
    step_order  INTEGER      NOT NULL,
    step_type   VARCHAR(50)  NOT NULL CHECK (step_type IN ('harvested', 'processed', 'stored', 'transported', 'delivered')),
    location    VARCHAR(255) NOT NULL,
    latitude    DECIMAL(9,6),
    longitude   DECIMAL(9,6),
    arrived_at  TIMESTAMPTZ  NOT NULL,
    departed_at TIMESTAMPTZ,
    notes       TEXT,
    UNIQUE (batch_id, step_order)
);

CREATE INDEX idx_journey_steps_batch_id ON journey_steps(batch_id);

-- Temperature readings
CREATE TABLE temperature_readings (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id       UUID        NOT NULL REFERENCES batches(id),
    recorded_at    TIMESTAMPTZ NOT NULL,
    temp_celsius   DECIMAL(4,1) NOT NULL,
    min_acceptable DECIMAL(4,1) NOT NULL,
    max_acceptable DECIMAL(4,1) NOT NULL,
    location       VARCHAR(255),
    CONSTRAINT chk_temp_range CHECK (min_acceptable < max_acceptable)
);

CREATE INDEX idx_temp_readings_batch_time ON temperature_readings(batch_id, recorded_at);

-- Quality checks
CREATE TABLE quality_checks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id   UUID         NOT NULL REFERENCES batches(id),
    check_type VARCHAR(100) NOT NULL,
    passed     BOOLEAN      NOT NULL,
    checked_at TIMESTAMPTZ  NOT NULL,
    inspector  VARCHAR(255),
    notes      TEXT
);

CREATE INDEX idx_quality_checks_batch_id ON quality_checks(batch_id);

-- Recalls
CREATE TABLE recalls (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id     UUID         NOT NULL UNIQUE REFERENCES batches(id),
    severity     VARCHAR(20)  NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    reason       TEXT         NOT NULL,
    instructions TEXT         NOT NULL,
    recalled_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE
);

-- Certifications
CREATE TABLE certifications (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id     UUID         NOT NULL REFERENCES batches(id),
    cert_type    VARCHAR(100) NOT NULL,
    issuing_body VARCHAR(255) NOT NULL,
    valid_until  DATE
);

CREATE INDEX idx_certifications_batch_id ON certifications(batch_id);

-- Sustainability
CREATE TABLE sustainability (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id     UUID          NOT NULL UNIQUE REFERENCES batches(id),
    co2_kg       DECIMAL(8,2),
    water_liters DECIMAL(10,2),
    transport_km DECIMAL(8,2)
);

-- Users
CREATE TABLE users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email        VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(100),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Complaints
CREATE TABLE complaints (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id       UUID        NOT NULL REFERENCES batches(id),
    user_id        UUID        NOT NULL REFERENCES users(id),
    complaint_type VARCHAR(50) NOT NULL CHECK (complaint_type IN ('taste_smell', 'packaging_damaged', 'foreign_object', 'suspected_spoilage', 'other')),
    description    TEXT,
    photo_url      VARCHAR(500),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_complaints_batch_id ON complaints(batch_id);
CREATE INDEX idx_complaints_user_id ON complaints(user_id);

-- Scan history
CREATE TABLE scan_history (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id),
    batch_id   UUID        NOT NULL REFERENCES batches(id),
    scanned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_scan_history_batch_id ON scan_history(batch_id);
CREATE INDEX idx_scan_history_user_id ON scan_history(user_id);
