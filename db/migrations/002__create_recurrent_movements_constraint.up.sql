ALTER TABLE movements
ADD COLUMN recurrent_id uuid;

ALTER TABLE movements
ADD CONSTRAINT movements_recurrent_id_fk
FOREIGN KEY (recurrent_id) REFERENCES recurrent_movements (id);