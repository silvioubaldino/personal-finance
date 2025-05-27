-- Make sub_category_id nullable in recurrent_movements table
ALTER TABLE recurrent_movements
    ALTER COLUMN sub_category_id DROP NOT NULL; 