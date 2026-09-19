CREATE TABLE IF NOT EXISTS aviones (
    id SERIAL PRIMARY KEY,
    nombre VARCHAR(100),
    asientos_regular INTEGER,
    asientos_vip INTEGER,
    fabricante VARCHAR(100)
);

CREATE TABLE IF NOT EXISTS ciudades (
    id SERIAL PRIMARY KEY,
    codigo VARCHAR(10),
    pais VARCHAR(100),
    region VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS puertas (
    id SERIAL PRIMARY KEY,
    puerta VARCHAR(10),
    id_ciudad INTEGER REFERENCES ciudades(id)
);

CREATE TABLE IF NOT EXISTS asientos (
    id SERIAL PRIMARY KEY,
    codigo VARCHAR(10),
    id_avion INTEGER REFERENCES aviones(id),
    estado VARCHAR(20),
    clase VARCHAR(20)
);

CREATE TABLE IF NOT EXISTS estados_vuelo (
    id SERIAL PRIMARY KEY,
    nombre VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS vuelos (
    id SERIAL PRIMARY KEY,
    id_origen INTEGER REFERENCES ciudades(id),
    id_destino INTEGER REFERENCES ciudades(id),
    id_estado_vuelo INTEGER REFERENCES estados_vuelo(id),
    id_puerta INTEGER REFERENCES puertas(id),
    id_avion INTEGER REFERENCES aviones(id),
    llegada_programada BIGINT,
    salida_programada BIGINT,
    llegada_real BIGINT,
    salida_real BIGINT,
    fecha_llegada BIGINT,
    fecha_salida BIGINT
);

CREATE TABLE IF NOT EXISTS boletos (
    id_boleto SERIAL PRIMARY KEY,
    nombre_pasajero VARCHAR(255),
    email_pasajero VARCHAR(255),
    id_vuelo INTEGER REFERENCES vuelos(id),
    id_asiento INTEGER REFERENCES asientos(id),
    costo DECIMAL(10, 2),
    tiempo_de_viaje INTEGER,
    pasaporte VARCHAR(50),
    estado VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS precios (
    id SERIAL PRIMARY KEY,
    matriz_precios_regular JSONB,
    matriz_precios_vip JSONB
);

CREATE TABLE IF NOT EXISTS detalles_vuelos (
    id SERIAL PRIMARY KEY,
    matriz_tiempos JSONB
);

-- Basic data
INSERT INTO estados_vuelo (nombre) VALUES 
('SCHEDULED'), ('BOARDING'), ('DEPARTED'), ('IN_FLIGHT'), ('LANDED'), ('ARRIVED'), ('CANCELLED'), ('DELAYED')
ON CONFLICT DO NOTHING;

-- 4 modelos de avion, 50 instancias fisicas (aircraft_id 1-50 del CSV)
-- IDs 1-12: A380, 13-25: Boeing 777, 26-38: A350, 39-50: Boeing 787
INSERT INTO aviones (nombre, asientos_regular, asientos_vip, fabricante) VALUES 
-- A380 instances (IDs 1-12)
('Airbus A380-800', 439, 10, 'Airbus'),
('Airbus A380-800', 439, 10, 'Airbus'),
('Airbus A380-800', 439, 10, 'Airbus'),
('Airbus A380-800', 439, 10, 'Airbus'),
('Airbus A380-800', 439, 10, 'Airbus'),
('Airbus A380-800', 439, 10, 'Airbus'),
('Airbus A380-800', 439, 10, 'Airbus'),
('Airbus A380-800', 439, 10, 'Airbus'),
('Airbus A380-800', 439, 10, 'Airbus'),
('Airbus A380-800', 439, 10, 'Airbus'),
('Airbus A380-800', 439, 10, 'Airbus'),
('Airbus A380-800', 439, 10, 'Airbus'),
-- Boeing 777 instances (IDs 13-25)
('Boeing 777-300ER', 300, 10, 'Boeing'),
('Boeing 777-300ER', 300, 10, 'Boeing'),
('Boeing 777-300ER', 300, 10, 'Boeing'),
('Boeing 777-300ER', 300, 10, 'Boeing'),
('Boeing 777-300ER', 300, 10, 'Boeing'),
('Boeing 777-300ER', 300, 10, 'Boeing'),
('Boeing 777-300ER', 300, 10, 'Boeing'),
('Boeing 777-300ER', 300, 10, 'Boeing'),
('Boeing 777-300ER', 300, 10, 'Boeing'),
('Boeing 777-300ER', 300, 10, 'Boeing'),
('Boeing 777-300ER', 300, 10, 'Boeing'),
('Boeing 777-300ER', 300, 10, 'Boeing'),
('Boeing 777-300ER', 300, 10, 'Boeing'),
-- A350 instances (IDs 26-38)
('Airbus A350-900', 250, 12, 'Airbus'),
('Airbus A350-900', 250, 12, 'Airbus'),
('Airbus A350-900', 250, 12, 'Airbus'),
('Airbus A350-900', 250, 12, 'Airbus'),
('Airbus A350-900', 250, 12, 'Airbus'),
('Airbus A350-900', 250, 12, 'Airbus'),
('Airbus A350-900', 250, 12, 'Airbus'),
('Airbus A350-900', 250, 12, 'Airbus'),
('Airbus A350-900', 250, 12, 'Airbus'),
('Airbus A350-900', 250, 12, 'Airbus'),
('Airbus A350-900', 250, 12, 'Airbus'),
('Airbus A350-900', 250, 12, 'Airbus'),
('Airbus A350-900', 250, 12, 'Airbus'),
-- Boeing 787 instances (IDs 39-50)
('Boeing 787-9 Dreamliner', 220, 8, 'Boeing'),
('Boeing 787-9 Dreamliner', 220, 8, 'Boeing'),
('Boeing 787-9 Dreamliner', 220, 8, 'Boeing'),
('Boeing 787-9 Dreamliner', 220, 8, 'Boeing'),
('Boeing 787-9 Dreamliner', 220, 8, 'Boeing'),
('Boeing 787-9 Dreamliner', 220, 8, 'Boeing'),
('Boeing 787-9 Dreamliner', 220, 8, 'Boeing'),
('Boeing 787-9 Dreamliner', 220, 8, 'Boeing'),
('Boeing 787-9 Dreamliner', 220, 8, 'Boeing'),
('Boeing 787-9 Dreamliner', 220, 8, 'Boeing'),
('Boeing 787-9 Dreamliner', 220, 8, 'Boeing'),
('Boeing 787-9 Dreamliner', 220, 8, 'Boeing')
ON CONFLICT DO NOTHING;

-- Ciudades representativas
INSERT INTO ciudades (codigo, pais, region) VALUES 
('ATL', 'Estados Unidos', 'America'),
('PEK', 'China', 'Asia'),
('DXB', 'Emiratos', 'Asia'),
('TYO', 'Japon', 'Asia'),
('LON', 'Reino Unido', 'Europa'),
('LAX', 'Estados Unidos', 'America'),
('PAR', 'Francia', 'Europa'),
('FRA', 'Alemania', 'Europa'),
('IST', 'Turquia', 'Europa'),
('SIN', 'Singapur', 'Asia'),
('MAD', 'España', 'Europa'),
('AMS', 'Países Bajos', 'Europa'),
('DFW', 'Estados Unidos', 'America'),
('CAN', 'China', 'Asia'),
('SAO', 'Brasil', 'America')
ON CONFLICT DO NOTHING;

-- Puertas en formato G1-G30 (igual al CSV del dataset)
INSERT INTO puertas (puerta, id_ciudad) VALUES 
('G1',  1), ('G2',  2), ('G3',  3), ('G4',  4), ('G5',  5),
('G6',  6), ('G7',  7), ('G8',  8), ('G9',  9), ('G10', 10),
('G11', 11), ('G12', 12), ('G13', 13), ('G14', 14), ('G15', 15),
('G16', 1), ('G17', 2), ('G18', 3), ('G19', 4), ('G20', 5),
('G21', 6), ('G22', 7), ('G23', 8), ('G24', 9), ('G25', 10),
('G26', 11), ('G27', 12), ('G28', 13), ('G29', 14), ('G30', 15)
ON CONFLICT DO NOTHING;
