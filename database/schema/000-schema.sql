CREATE TABLE IF NOT EXISTS users
(
    id       SERIAL PRIMARY KEY,
    email    VARCHAR(200) UNIQUE NOT NULL,
    name     VARCHAR(200)        NOT NULL,
    role     VARCHAR(200)        NOT NULL,
    password VARCHAR(200)        NOT NULL,
    address  VARCHAR(200),
    city     VARCHAR(200),
    state    VARCHAR(200),
    country  VARCHAR(200),
    zip_code VARCHAR(200)
);

CREATE TABLE IF NOT EXISTS sessions
(
    id         SERIAL PRIMARY KEY,
    token      VARCHAR(200) UNIQUE NOT NULL,
    user_id    INTEGER             NOT NULL REFERENCES users,
    created_at TIMESTAMP           NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP           NOT NULL
);

CREATE TABLE IF NOT EXISTS categories
(
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(200) NOT NULL,
    description TEXT         NOT NULL
);

CREATE TABLE IF NOT EXISTS products
(
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(200) NOT NULL UNIQUE,
    description   TEXT         NOT NULL,
    category_id   INTEGER      NOT NULL REFERENCES categories,
    extension     VARCHAR(200) NOT NULL,
    enabled       BOOLEAN      NOT NULl,
    pricing       JSONB        NOT NULL,
    settings      JSONB        NOT NULL,
    stock         INTEGER      NOT NULL,
    stock_control INTEGER      NOT NULl
);

CREATE TABLE IF NOT EXISTS product_options
(
    product_id   INTEGER      NOT NULL REFERENCES products,
    name         VARCHAR(200) NOT NULL,
    display_name VARCHAR(200) NOT NULL,
    type         VARCHAR(200) NOT NULL,
    regex        TEXT         NOT NULL,
    values       JSONB        NOT NULL,
    description  TEXT         NOT NULl,
    PRIMARY KEY (product_id, name)
);


CREATE TABLE IF NOT EXISTS services
(
    id                  SERIAL PRIMARY KEY,
    label               VARCHAR(200)   NOT NULL,
    user_id             INTEGER        NOT NULL REFERENCES users,
    status              VARCHAR(200)   NOT NULL,
    cancellation_reason TEXT,
    billing_cycle       INTEGER        NOT NULL,
    price               DECIMAL(12, 2) NOT NULL,
    extension           VARCHAR(200)   NOT NULL,
    settings            JSONB          NOT NULL,
    expires_at          TIMESTAMP      NOT NULL,
    created_at          TIMESTAMP      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    cancelled_at        TIMESTAMP
);

CREATE TABLE IF NOT EXISTS invoices
(
    id                  SERIAL PRIMARY KEY,
    user_id             INTEGER        NOT NULL REFERENCES users,
    status              VARCHAR(200)   NOT NULL,
    cancellation_reason TEXT,
    paid_at             TIMESTAMP,
    due_at              TIMESTAMP      NOT NULL,
    amount              DECIMAL(12, 2) NOT NULL,
    created_at          TIMESTAMP      NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS invoice_items
(
    id          SERIAL PRIMARY KEY,
    invoice_id  INTEGER        NOT NULL REFERENCES invoices,
    description TEXT           NOT NULL,
    amount      DECIMAL(12, 2) NOT NULL,
    type        VARCHAR(200)   NOT NULL,
    item_id     INTEGER,
    created_at  TIMESTAMP      NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS gateways
(
    id           SERIAL PRIMARY KEY,
    display_name VARCHAR(200) NOT NULL UNIQUE,
    name         VARCHAR(200) NOT NULL UNIQUE,
    settings     JSONB        NOT NULL,
    enabled      BOOLEAN      NOT NULL,
    fee          VARCHAR(200)
);


CREATE TABLE IF NOT EXISTS invoice_payments
(
    id           SERIAL PRIMARY KEY,
    invoice_id   INTEGER        NOT NULL REFERENCES invoices,
    created_at   TIMESTAMP      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    description  VARCHAR(200)   NOT NULL,
    amount       DECIMAL(12, 2) NOT NULL,
    reference_id VARCHAR(200)   NOT NULL,
    gateway      VARCHAR(200)   NOT NULL
);

CREATE TABLE IF NOT EXISTS servers
(
    id        SERiAL PRIMARY KEY,
    label     VARCHAR(200) NOT NULL,
    extension VARCHAR(200) NOT NULL,
    settings  JSONB
);

CREATE TABLE IF NOT EXISTS settings
(
    id    SERIAL PRIMARY KEY,
    key   VARCHAR(200) UNIQUE NOT NULL,
    value TEXT         NOT NULL
);