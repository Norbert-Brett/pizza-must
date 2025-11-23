-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL CHECK (price >= 0),
    category_id UUID NOT NULL,
    image_url VARCHAR(500),
    stock INTEGER NOT NULL DEFAULT 0 CHECK (stock >= 0),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_products_category
        FOREIGN KEY (category_id)
        REFERENCES categories(id)
        ON DELETE RESTRICT
);

-- Create index on category_id for filtering by category
CREATE INDEX idx_products_category_id ON products(category_id);

-- Create index on name for search functionality
CREATE INDEX idx_products_name ON products(name);

-- Create full-text search index for name and description
CREATE INDEX idx_products_search ON products USING gin(to_tsvector('english', name || ' ' || COALESCE(description, '')));

-- Create index on price for sorting
CREATE INDEX idx_products_price ON products(price);

-- Create index on created_at for sorting
CREATE INDEX idx_products_created_at ON products(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_products_created_at;
DROP INDEX IF EXISTS idx_products_price;
DROP INDEX IF EXISTS idx_products_search;
DROP INDEX IF EXISTS idx_products_name;
DROP INDEX IF EXISTS idx_products_category_id;
DROP TABLE IF EXISTS products;
-- +goose StatementEnd
