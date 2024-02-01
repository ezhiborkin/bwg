CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    wallet_id VARCHAR(255) NOT NULL,
    amount FLOAT NOT NULL,
    type VARCHAR(255) NOT NULL,
    date_created TIMESTAMP,
    status VARCHAR(255) NOT NULL
);
