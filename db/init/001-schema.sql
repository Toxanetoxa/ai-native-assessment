SET NAMES utf8mb4;

CREATE TABLE dataset_metadata (
  id TINYINT NOT NULL PRIMARY KEY CHECK (id = 1),
  version VARCHAR(64) NOT NULL,
  as_of DATETIME NOT NULL,
  timezone VARCHAR(64) NOT NULL,
  currency CHAR(3) NOT NULL,
  default_from DATE NOT NULL,
  default_to DATE NOT NULL,
  source_hashes JSON NOT NULL
) ENGINE=InnoDB;

CREATE TABLE suppliers (
  id VARCHAR(32) PRIMARY KEY,
  name VARCHAR(255) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE supplier_policies (
  id VARCHAR(32) PRIMARY KEY,
  supplier_id VARCHAR(32) NOT NULL,
  valid_from DATE NOT NULL,
  valid_to DATE NULL,
  confirmed BOOLEAN NOT NULL,
  window_days INT NULL CHECK (window_days IS NULL OR window_days > 0),
  starts_at VARCHAR(64) NULL,
  ends_at VARCHAR(64) NULL,
  include_first_day BOOLEAN NULL,
  calendar VARCHAR(32) NULL,
  requires_packaging BOOLEAN NULL,
  accepts_good_quality BOOLEAN NULL,
  CONSTRAINT fk_policy_supplier FOREIGN KEY (supplier_id) REFERENCES suppliers(id),
  INDEX idx_policies_supplier_dates (supplier_id, valid_from, valid_to)
) ENGINE=InnoDB;

CREATE TABLE return_lines (
  id VARCHAR(32) PRIMARY KEY,
  return_id VARCHAR(32) NOT NULL,
  supplier_id VARCHAR(32) NOT NULL,
  policy_id VARCHAR(32) NOT NULL,
  customer_id VARCHAR(32) NOT NULL,
  product VARCHAR(255) NOT NULL,
  quantity INT NOT NULL CHECK (quantity > 0),
  unit_book_cost_minor BIGINT NOT NULL CHECK (unit_book_cost_minor >= 0),
  supplier_shipped_at DATETIME NULL,
  requested_at DATETIME NULL,
  warehouse_received_at DATETIME NOT NULL,
  packaging_complete BOOLEAN NULL,
  quality VARCHAR(32) NOT NULL,
  posted BOOLEAN NOT NULL,
  CONSTRAINT fk_return_supplier FOREIGN KEY (supplier_id) REFERENCES suppliers(id),
  CONSTRAINT fk_return_policy FOREIGN KEY (policy_id) REFERENCES supplier_policies(id),
  INDEX idx_returns_document (return_id),
  INDEX idx_returns_received (warehouse_received_at, supplier_id),
  INDEX idx_returns_customer (customer_id)
) ENGINE=InnoDB;

CREATE TABLE supplier_returns (
  id VARCHAR(32) PRIMARY KEY,
  supplier_id VARCHAR(32) NOT NULL,
  dispatched_at DATETIME NOT NULL,
  posted BOOLEAN NOT NULL,
  CONSTRAINT fk_supplier_return_supplier FOREIGN KEY (supplier_id) REFERENCES suppliers(id),
  INDEX idx_supplier_returns_dispatched (dispatched_at, supplier_id)
) ENGINE=InnoDB;

CREATE TABLE supplier_return_lines (
  id VARCHAR(32) PRIMARY KEY,
  supplier_return_id VARCHAR(32) NOT NULL,
  return_line_id VARCHAR(32) NOT NULL,
  quantity INT NOT NULL CHECK (quantity > 0),
  CONSTRAINT fk_allocation_shipment FOREIGN KEY (supplier_return_id) REFERENCES supplier_returns(id),
  CONSTRAINT fk_allocation_return FOREIGN KEY (return_line_id) REFERENCES return_lines(id)
) ENGINE=InnoDB;

CREATE TABLE return_events (
  delivery_id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  event_id VARCHAR(32) NOT NULL,
  allocation_id VARCHAR(32) NOT NULL,
  type VARCHAR(32) NOT NULL,
  occurred_at DATETIME NOT NULL,
  recorded_at DATETIME NOT NULL,
  quantity INT NOT NULL CHECK (quantity >= 0),
  reason VARCHAR(32) NULL,
  reverses_event_id VARCHAR(32) NULL,
  CONSTRAINT fk_event_allocation FOREIGN KEY (allocation_id) REFERENCES supplier_return_lines(id),
  INDEX idx_events_identity (event_id),
  INDEX idx_events_observation (recorded_at, occurred_at),
  INDEX idx_events_allocation (allocation_id, occurred_at),
  INDEX idx_events_reversal (reverses_event_id)
) ENGINE=InnoDB;

CREATE TABLE stock_movements (
  id VARCHAR(32) PRIMARY KEY,
  return_line_id VARCHAR(32) NOT NULL,
  quantity_delta INT NOT NULL,
  value_delta_minor BIGINT NOT NULL,
  occurred_at DATETIME NOT NULL,
  recorded_at DATETIME NOT NULL,
  CONSTRAINT fk_stock_return FOREIGN KEY (return_line_id) REFERENCES return_lines(id),
  INDEX idx_stock_line_time (return_line_id, occurred_at),
  INDEX idx_stock_observation (recorded_at, occurred_at)
) ENGINE=InnoDB;

CREATE TABLE financial_entries (
  id VARCHAR(32) PRIMARY KEY,
  return_line_id VARCHAR(32) NOT NULL,
  type VARCHAR(32) NOT NULL,
  amount_minor BIGINT NOT NULL CHECK (amount_minor >= 0),
  occurred_at DATETIME NOT NULL,
  CONSTRAINT fk_financial_return FOREIGN KEY (return_line_id) REFERENCES return_lines(id),
  INDEX idx_financial_line_time (return_line_id, occurred_at)
) ENGINE=InnoDB;

CREATE TABLE shipments (
  id VARCHAR(32) PRIMARY KEY,
  supplier_id VARCHAR(32) NOT NULL,
  shipped_at DATETIME NOT NULL,
  quantity INT NOT NULL CHECK (quantity > 0),
  amount_minor BIGINT NOT NULL CHECK (amount_minor >= 0),
  CONSTRAINT fk_shipment_supplier FOREIGN KEY (supplier_id) REFERENCES suppliers(id),
  INDEX idx_shipments_date_supplier (shipped_at, supplier_id)
) ENGINE=InnoDB;
