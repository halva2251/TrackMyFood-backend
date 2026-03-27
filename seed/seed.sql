-- Food Flight Tracker - Demo Seed Data
-- 4 scenarios with deterministic UUIDs for reproducibility

-- ============================================================
-- PRODUCERS
-- ============================================================
INSERT INTO producers (id, name, location, country) VALUES
  ('00000000-0000-0000-0000-000000000001', 'Bio Hof Thurgau',     'Frauenfeld, Thurgau',  'CH'),
  ('00000000-0000-0000-0000-000000000002', 'Nordic Fish AS',       'Bergen, Hordaland',    'NO'),
  ('00000000-0000-0000-0000-000000000003', 'Alpina Dairy GmbH',   'Luzern, Zentralschweiz','CH'),
  ('00000000-0000-0000-0000-000000000004', 'Imkerei Sonnenberg',  'Appenzell, Appenzell',  'CH'),
  ('00000000-0000-0000-0000-000000000005', 'El Tony Mate',        'Lucerne, Lucerne',      'CH');

-- ============================================================
-- PRODUCTS
-- ============================================================
INSERT INTO products (id, producer_id, name, category, barcode, min_temp_celsius, max_temp_celsius, optimal_shelf_hours, optimal_handling_steps) VALUES
  ('00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0000-000000000001', 'Organic Strawberries 500g',  'fruits',  '7610000000001', 1.0, 4.0,  24, 3),
  ('00000000-0000-0000-0001-000000000002', '00000000-0000-0000-0000-000000000002', 'Atlantic Salmon Fillet 300g','seafood', '7610000000002', 0.0, 4.0,  48, 3),
  ('00000000-0000-0000-0001-000000000003', '00000000-0000-0000-0000-000000000003', 'Natural Yogurt 500g',       'dairy',   '7610000000003', 2.0, 6.0,  72, 3),
  ('00000000-0000-0000-0001-000000000004', '00000000-0000-0000-0000-000000000004', 'Mountain Flower Honey 250g','honey',   '7610000000004', 10.0, 25.0, 720, 2),
  ('00000000-0000-0000-0001-000000000005', '00000000-0000-0000-0000-000000000005', 'El Tony Mate 33cl',         'beverages','7640150491001', 2.0, 25.0, 8760, 3);

-- ============================================================
-- BATCHES (trust scores pre-calculated)
-- ============================================================
INSERT INTO batches (id, product_id, lot_number, production_date, expiry_date, trust_score, sub_score_cold_chain, sub_score_quality, sub_score_time_to_shelf, sub_score_producer, sub_score_handling, score_calculated_at) VALUES
  -- Scenario 1: Perfect batch (strawberries) ~94
  ('00000000-0000-0000-0002-000000000001', '00000000-0000-0000-0001-000000000001', 'LOT-2026-0312-A', '2026-03-12 06:00:00+01', '2026-03-28 23:59:59+01',
   94.00, 100.00, 100.00, 90.00, 85.00, 90.00, '2026-03-12 12:00:00+01'),

  -- Scenario 2: Sketchy batch (salmon) ~52
  ('00000000-0000-0000-0002-000000000002', '00000000-0000-0000-0001-000000000002', 'LOT-2026-0308-B', '2026-03-08 04:00:00+01', '2026-03-30 23:59:59+01',
   52.00, 40.00, 66.67, 60.00, 55.00, 40.00, '2026-03-10 10:00:00+01'),

  -- Scenario 3: Recalled product (yogurt) -> score 0
  ('00000000-0000-0000-0002-000000000003', '00000000-0000-0000-0001-000000000003', 'LOT-2026-0305-C', '2026-03-05 10:00:00+01', '2026-03-25 23:59:59+01',
   0.00, 100.00, 66.67, 85.00, 70.00, 100.00, '2026-03-07 09:00:00+01'),

  -- Scenario 4: Sustainable choice (honey) ~88
  ('00000000-0000-0000-0002-000000000004', '00000000-0000-0000-0001-000000000004', 'LOT-2026-0301-D', '2026-03-01 08:00:00+01', '2027-03-01 23:59:59+01',
   88.00, 100.00, 100.00, 95.00, 80.00, 100.00, '2026-03-02 10:00:00+01'),

  -- Scenario 5: El Tony Mate (from photo)
  ('00000000-0000-0000-0002-000000000005', '00000000-0000-0000-0001-000000000005', 'L2506347', '2025-05-18 16:38:00+01', '2027-05-18 23:59:59+01',
   92.00, 100.00, 100.00, 85.00, 90.00, 80.00, '2025-05-19 10:00:00+01');

-- ============================================================
-- JOURNEY STEPS
-- ============================================================

-- Scenario 1: Strawberries — 3 steps, short chain
INSERT INTO journey_steps (id, batch_id, step_order, step_type, location, latitude, longitude, arrived_at, departed_at) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', 1, 'harvested',   'Bio Hof, Frauenfeld',          47.5535, 8.8987, '2026-03-12 06:00:00+01', '2026-03-12 08:00:00+01'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', 2, 'stored',      'Cold Warehouse, Zurich',       47.3769, 8.5417, '2026-03-12 09:00:00+01', '2026-03-12 18:00:00+01'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', 3, 'delivered',   'Migros Zurich HB',             47.3783, 8.5403, '2026-03-12 19:00:00+01', NULL);

-- Scenario 2: Salmon — 5 steps, long chain
INSERT INTO journey_steps (id, batch_id, step_order, step_type, location, latitude, longitude, arrived_at, departed_at) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', 1, 'harvested',   'Fish Farm, Bergen',            60.3913, 5.3221, '2026-03-08 04:00:00+01', '2026-03-08 06:00:00+01'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', 2, 'processed',   'Processing Plant, Bergen',     60.3930, 5.3340, '2026-03-08 07:00:00+01', '2026-03-08 14:00:00+01'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', 3, 'stored',      'Cold Storage, Oslo',           59.9139, 10.7522,'2026-03-08 22:00:00+01', '2026-03-09 06:00:00+01'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', 4, 'transported', 'Truck Transit, Oslo to Zurich',NULL,     NULL,   '2026-03-09 07:00:00+01', '2026-03-10 03:00:00+01'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', 5, 'delivered',   'Coop Zurich Oerlikon',         47.4111, 8.5448, '2026-03-10 08:00:00+01', NULL);

-- Scenario 3: Yogurt — 3 steps
INSERT INTO journey_steps (id, batch_id, step_order, step_type, location, latitude, longitude, arrived_at, departed_at) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', 1, 'processed',   'Alpina Dairy, Luzern',         47.0502, 8.3093, '2026-03-05 10:00:00+01', '2026-03-05 18:00:00+01'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', 2, 'stored',      'Cold Warehouse, Basel',        47.5596, 7.5886, '2026-03-06 02:00:00+01', '2026-03-06 14:00:00+01'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', 3, 'delivered',   'Migros Basel SBB',             47.5476, 7.5898, '2026-03-06 16:00:00+01', NULL);

-- Scenario 4: Honey — 2 steps, ultra-short
INSERT INTO journey_steps (id, batch_id, step_order, step_type, location, latitude, longitude, arrived_at, departed_at) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000004', 1, 'harvested',   'Imkerei Sonnenberg, Appenzell',47.3307, 9.4092, '2026-03-01 08:00:00+01', '2026-03-01 12:00:00+01'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000004', 2, 'delivered',   'Hofladen Appenzell',           47.3312, 9.4088, '2026-03-01 14:00:00+01', NULL);

-- Scenario 5: El Tony Mate
INSERT INTO journey_steps (id, batch_id, step_order, step_type, location, latitude, longitude, arrived_at, departed_at) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000005', 1, 'processed',   'El Tony Production, Lucerne',  47.0502, 8.3093, '2025-05-18 16:38:00+01', '2025-05-19 08:00:00+01'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000005', 2, 'delivered',   'Coop City St. Annahof, Zurich', 47.3735, 8.5385, '2025-05-20 09:00:00+01', NULL);

-- ============================================================
-- TEMPERATURE READINGS
-- ============================================================

-- Scenario 1: Strawberries — 12 readings, all in range (1-4°C)
INSERT INTO temperature_readings (id, batch_id, recorded_at, temp_celsius, min_acceptable, max_acceptable, location) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', '2026-03-12 08:00:00+01', 2.1, 1.0, 4.0, 'Transport to warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', '2026-03-12 09:00:00+01', 1.8, 1.0, 4.0, 'Cold Warehouse intake'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', '2026-03-12 10:00:00+01', 2.0, 1.0, 4.0, 'Cold Warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', '2026-03-12 11:00:00+01', 2.2, 1.0, 4.0, 'Cold Warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', '2026-03-12 12:00:00+01', 1.9, 1.0, 4.0, 'Cold Warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', '2026-03-12 13:00:00+01', 2.3, 1.0, 4.0, 'Cold Warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', '2026-03-12 14:00:00+01', 2.1, 1.0, 4.0, 'Cold Warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', '2026-03-12 15:00:00+01', 2.4, 1.0, 4.0, 'Cold Warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', '2026-03-12 16:00:00+01', 2.0, 1.0, 4.0, 'Cold Warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', '2026-03-12 17:00:00+01', 2.5, 1.0, 4.0, 'Cold Warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', '2026-03-12 18:00:00+01', 3.0, 1.0, 4.0, 'Transport to store'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', '2026-03-12 19:00:00+01', 3.2, 1.0, 4.0, 'Store delivery');

-- Scenario 2: Salmon — 24 readings, 4 spike during transport (cold chain breach)
INSERT INTO temperature_readings (id, batch_id, recorded_at, temp_celsius, min_acceptable, max_acceptable, location) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-08 06:00:00+01', 1.2, 0.0, 4.0, 'Bergen facility'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-08 08:00:00+01', 1.5, 0.0, 4.0, 'Processing plant'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-08 10:00:00+01', 1.8, 0.0, 4.0, 'Processing plant'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-08 12:00:00+01', 2.0, 0.0, 4.0, 'Processing plant'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-08 14:00:00+01', 1.6, 0.0, 4.0, 'Transport to Oslo'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-08 16:00:00+01', 2.1, 0.0, 4.0, 'Transport to Oslo'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-08 18:00:00+01', 2.5, 0.0, 4.0, 'Transport to Oslo'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-08 20:00:00+01', 3.0, 0.0, 4.0, 'Transport to Oslo'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-08 22:00:00+01', 1.5, 0.0, 4.0, 'Oslo cold storage'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-09 00:00:00+01', 1.2, 0.0, 4.0, 'Oslo cold storage'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-09 02:00:00+01', 1.0, 0.0, 4.0, 'Oslo cold storage'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-09 04:00:00+01', 1.3, 0.0, 4.0, 'Oslo cold storage'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-09 06:00:00+01', 2.0, 0.0, 4.0, 'Loading truck'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-09 08:00:00+01', 3.5, 0.0, 4.0, 'Truck transit'),
  -- COLD CHAIN BREACH: 4 readings above 4°C
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-09 10:00:00+01', 6.5,  0.0, 4.0, 'Truck transit - BREACH'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-09 12:00:00+01', 8.2,  0.0, 4.0, 'Truck transit - BREACH'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-09 14:00:00+01', 11.5, 0.0, 4.0, 'Truck transit - BREACH'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-09 16:00:00+01', 9.0,  0.0, 4.0, 'Truck transit - BREACH'),
  -- Recovery
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-09 18:00:00+01', 5.0, 0.0, 4.0, 'Truck transit - recovering'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-09 20:00:00+01', 3.8, 0.0, 4.0, 'Truck transit'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-09 22:00:00+01', 3.0, 0.0, 4.0, 'Approaching Zurich'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-10 00:00:00+01', 2.5, 0.0, 4.0, 'Zurich unloading'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-10 04:00:00+01', 1.8, 0.0, 4.0, 'Store cold storage'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', '2026-03-10 08:00:00+01', 1.5, 0.0, 4.0, 'Store shelf');

-- Scenario 3: Yogurt — 18 readings, all in range
INSERT INTO temperature_readings (id, batch_id, recorded_at, temp_celsius, min_acceptable, max_acceptable, location) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-05 10:00:00+01', 3.0, 2.0, 6.0, 'Dairy facility'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-05 12:00:00+01', 3.2, 2.0, 6.0, 'Dairy facility'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-05 14:00:00+01', 3.5, 2.0, 6.0, 'Dairy facility'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-05 16:00:00+01', 3.8, 2.0, 6.0, 'Packaging'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-05 18:00:00+01', 4.0, 2.0, 6.0, 'Loading'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-05 20:00:00+01', 4.2, 2.0, 6.0, 'Transport'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-05 22:00:00+01', 4.0, 2.0, 6.0, 'Transport'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-06 00:00:00+01', 3.8, 2.0, 6.0, 'Transport'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-06 02:00:00+01', 3.5, 2.0, 6.0, 'Basel warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-06 04:00:00+01', 3.2, 2.0, 6.0, 'Basel warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-06 06:00:00+01', 3.0, 2.0, 6.0, 'Basel warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-06 08:00:00+01', 3.1, 2.0, 6.0, 'Basel warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-06 10:00:00+01', 3.3, 2.0, 6.0, 'Basel warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-06 12:00:00+01', 3.5, 2.0, 6.0, 'Basel warehouse'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-06 14:00:00+01', 3.8, 2.0, 6.0, 'Loading for store'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-06 15:00:00+01', 4.0, 2.0, 6.0, 'Transport to store'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-06 16:00:00+01', 4.2, 2.0, 6.0, 'Store delivery'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', '2026-03-06 17:00:00+01', 3.9, 2.0, 6.0, 'Store shelf');

-- Scenario 4: Honey — 6 readings, all in range (10-25°C, ambient)
INSERT INTO temperature_readings (id, batch_id, recorded_at, temp_celsius, min_acceptable, max_acceptable, location) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000004', '2026-03-01 08:00:00+01', 15.0, 10.0, 25.0, 'Apiary'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000004', '2026-03-01 09:00:00+01', 16.2, 10.0, 25.0, 'Apiary packaging'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000004', '2026-03-01 10:00:00+01', 17.0, 10.0, 25.0, 'Apiary storage'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000004', '2026-03-01 11:00:00+01', 18.5, 10.0, 25.0, 'Transport'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000004', '2026-03-01 12:00:00+01', 19.0, 10.0, 25.0, 'Transport'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000004', '2026-03-01 14:00:00+01', 20.0, 10.0, 25.0, 'Hofladen shelf');

-- ============================================================
-- QUALITY CHECKS
-- ============================================================

-- Scenario 1: Strawberries — 3 checks, all passed
INSERT INTO quality_checks (id, batch_id, check_type, passed, checked_at, inspector, notes) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', 'visual_inspection',  true,  '2026-03-12 06:30:00+01', 'Hans Mueller',   'Fresh, no bruising'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', 'packaging_check',    true,  '2026-03-12 08:00:00+01', 'Anna Schmidt',   'Sealed correctly'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', 'temp_verification',  true,  '2026-03-12 19:00:00+01', 'Store receiver', 'Temperature within range at delivery');

-- Scenario 2: Salmon — 3 checks, 1 failed
INSERT INTO quality_checks (id, batch_id, check_type, passed, checked_at, inspector, notes) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', 'visual_inspection',  true,  '2026-03-08 07:00:00+01', 'Erik Larsen',     'Good color and texture'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', 'lab_test',           true,  '2026-03-08 10:00:00+01', 'Lab Bergen',      'Bacterial levels normal'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', 'temp_verification',  false, '2026-03-10 08:00:00+01', 'Store receiver',  'Temperature log shows sustained breach during transit');

-- Scenario 3: Yogurt — 3 checks, 1 failed (lab test)
INSERT INTO quality_checks (id, batch_id, check_type, passed, checked_at, inspector, notes) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', 'visual_inspection',  true,  '2026-03-05 11:00:00+01', 'Maria Weber',    'Normal appearance'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', 'packaging_check',    true,  '2026-03-05 17:00:00+01', 'Peter Keller',   'Sealed and labeled correctly'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', 'lab_test',           false, '2026-03-07 08:00:00+01', 'Lab Luzern',     'Listeria monocytogenes detected above threshold');

-- Scenario 4: Honey — 2 checks, all passed
INSERT INTO quality_checks (id, batch_id, check_type, passed, checked_at, inspector, notes) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000004', 'visual_inspection',  true,  '2026-03-01 08:30:00+01', 'Josef Manser',   'Clear golden color, no crystallization'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000004', 'lab_test',           true,  '2026-03-01 10:00:00+01', 'Lab Appenzell',  'Moisture content 17.2%, all parameters within spec');

-- ============================================================
-- RECALLS (only yogurt)
-- ============================================================
INSERT INTO recalls (id, batch_id, severity, reason, instructions, recalled_at, is_active) VALUES
  ('00000000-0000-0000-0003-000000000001', '00000000-0000-0000-0002-000000000003', 'critical',
   'Listeria monocytogenes detected in routine lab testing. Contamination traced to packaging line.',
   'Do not consume this product. Return to your point of purchase for a full refund. If you have consumed this product and feel unwell, contact your doctor immediately.',
   '2026-03-07 10:00:00+01', true);

-- ============================================================
-- CERTIFICATIONS
-- ============================================================

-- Scenario 1: Strawberries — Bio Suisse
INSERT INTO certifications (id, batch_id, cert_type, issuing_body, valid_until) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', 'bio', 'Bio Suisse', '2027-01-01');

-- Scenario 4: Honey — Bio + Fair Trade
INSERT INTO certifications (id, batch_id, cert_type, issuing_body, valid_until) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000004', 'bio',        'Bio Suisse',        '2027-01-01'),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000004', 'fair_trade', 'Fairtrade International', '2027-06-01');

-- ============================================================
-- SUSTAINABILITY
-- ============================================================
INSERT INTO sustainability (id, batch_id, co2_kg, water_liters, transport_km) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000001', 0.80,   120.00,  45.00),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000002', 4.20,  2800.00, 1850.00),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000003', 1.50,   500.00,  120.00),
  (gen_random_uuid(), '00000000-0000-0000-0002-000000000004', 0.30,    45.00,   12.00);

-- ============================================================
-- USERS (demo accounts)
-- ============================================================
-- password for both demo users is "demo123"
INSERT INTO users (id, email, display_name, password_hash) VALUES
  ('00000000-0000-0000-0004-000000000001', 'demo@trackmyfood.ch',    'Demo User',  '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'),
  ('00000000-0000-0000-0004-000000000002', 'tester@trackmyfood.ch',  'Test User',  '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy');

-- ============================================================
-- SCAN HISTORY (User 2 scanned the yogurt — will be affected by recall)
-- ============================================================
INSERT INTO scan_history (id, user_id, batch_id, scanned_at) VALUES
  (gen_random_uuid(), '00000000-0000-0000-0004-000000000002', '00000000-0000-0000-0002-000000000003', '2026-03-06 18:00:00+01');
