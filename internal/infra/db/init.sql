CREATE TABLE deposits (
    id              BIGSERIAL PRIMARY KEY,
    tx_hash         VARCHAR(66) NOT NULL,
    block_number    BIGINT NOT NULL,
    token_address   VARCHAR(42) NOT NULL,
    from_address    VARCHAR(42) NOT NULL,
    to_address      VARCHAR(42) NOT NULL,
    amount          NUMERIC(78, 0) NOT NULL,
    status          VARCHAR(16) NOT NULL,
    created_at      TIMESTAMP DEFAULT now(),
    confirmed_at    TIMESTAMP
    revertied_at    TIMESTAMP
);
CREATE UNIQUE INDEX uniq_deposit_tx
ON deposits (tx_hash, token_address);


CREATE TABLE ledger_entries (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    asset           VARCHAR(32) NOT NULL,
    amount          NUMERIC(78, 0) NOT NULL,
    entry_type      VARCHAR(32) NOT NULL,
    ref_id          BIGINT,
    created_at      TIMESTAMP DEFAULT now()
);
CREATE INDEX idx_ledger_user_asset
ON ledger_entries (user_id, asset);


CREATE TABLE balances (
    user_id BIGINT NOT NULL,
    asset   VARCHAR(32) NOT NULL,
    balance NUMERIC(78, 0) NOT NULL,
    PRIMARY KEY (user_id, asset)
);


CREATE TABLE deposit_addresses (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    chain           VARCHAR(32) NOT NULL,
    asset           VARCHAR(32) NOT NULL,
    address         VARCHAR(42) NOT NULL,
    created_at      TIMESTAMP DEFAULT now(),
    UNIQUE (chain, address)
);


CREATE TABLE blocks (
    number      BIGINT PRIMARY KEY,
    hash        VARCHAR(66) NOT NULL,
    parent_hash VARCHAR(66) NOT NULL
);


CREATE TABLE withdraws (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    asset           VARCHAR(32) NOT NULL,
    amount          NUMERIC(78,0) NOT NULL,
    to_address      VARCHAR(42) NOT NULL,
    status          VARCHAR(32) NOT NULL,
    tx_hash         VARCHAR(66),
    created_at      TIMESTAMP DEFAULT now(),
    confirmed_at    TIMESTAMP
);
