-- Table 1: wallets
CREATE TABLE wallets (
    id VARCHAR(255) PRIMARY KEY,
    account_id VARCHAR(255) UNIQUE
);

-- Table 2: subwallets
CREATE TABLE subwallets (
    id SERIAL PRIMARY KEY,
    wallet_id VARCHAR(255),
    currency VARCHAR(255) NOT NULL,
    amount FLOAT NOT NULL,
    frozen_amount FLOAT,
    FOREIGN KEY (wallet_id) REFERENCES wallets(id)
);

-- Table 3: transactions
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    wallet_id VARCHAR(255) NOT NULL,
    amount FLOAT NOT NULL,
    type VARCHAR(255) NOT NULL,
    date_created TIMESTAMP,
    status VARCHAR(255) NOT NULL,
    FOREIGN KEY (wallet_id) REFERENCES wallets(id)
);