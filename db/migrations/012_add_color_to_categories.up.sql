ALTER TABLE categories ADD COLUMN color VARCHAR(7);

-- Update default categories with colors
UPDATE categories SET color = '#FF5733' WHERE id = '45b027f4-1f24-4cc9-9719-ce5f51d2d624'; -- Alimentação
UPDATE categories SET color = '#33FF57' WHERE id = 'a35ddc4b-d217-48e3-a5cc-e669fc3afaba'; -- Bem-estar
UPDATE categories SET color = '#FF3333' WHERE id = 'b0fe3149-0d12-407d-b094-e1b7f7e2c79c'; -- Dívidas
UPDATE categories SET color = '#3357FF' WHERE id = '919acdb5-ab4b-499c-8cf7-10985dd5abd1'; -- Doação
UPDATE categories SET color = '#33FFF5' WHERE id = '1ea04b90-eb7b-40b5-944a-c9778bc9a58b'; -- Educação
UPDATE categories SET color = '#808080' WHERE id = '89d7f952-5b08-49d4-9bb3-306bb25504dd'; -- Impostos
UPDATE categories SET color = '#228B22' WHERE id = 'c039c49d-7ef7-4b11-9f15-6187732cc57d'; -- Investimentos
UPDATE categories SET color = '#FFC300' WHERE id = '188b908e-c711-473d-8850-043ccb76a2b8'; -- Lazer
UPDATE categories SET color = '#800080' WHERE id = '5e83a71d-f52f-4339-83d5-860fb97ca786'; -- Moradia
UPDATE categories SET color = '#A52A2A' WHERE id = '5839cf02-edfb-44a6-aa8d-061d7d2ec0b3'; -- Pet
UPDATE categories SET color = '#FF1493' WHERE id = '000314b6-e998-41a5-9a19-c64283114cf7'; -- Saúde
UPDATE categories SET color = '#00BFFF' WHERE id = '53beefec-bd9b-423d-b5a3-9e10c2cede90'; -- Streaming
UPDATE categories SET color = '#FF4500' WHERE id = 'f672d741-d64e-45af-bf58-ef37294019ae'; -- Supermercado
UPDATE categories SET color = '#708090' WHERE id = '1d50405e-c42a-4991-b480-bc628d9b8713'; -- Taxas
UPDATE categories SET color = '#4682B4' WHERE id = '0689e288-f45d-43bb-aad5-560466d2715b'; -- Transporte
UPDATE categories SET color = '#00CED1' WHERE id = 'b1ee772c-e072-4b13-9cc0-6ff414b5cea6'; -- Viagens
UPDATE categories SET color = '#32CD32' WHERE id = 'f572d94e-648e-45d6-a0a4-865a6b9157fe'; -- Remuneração

