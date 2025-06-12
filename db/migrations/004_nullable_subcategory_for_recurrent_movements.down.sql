-- Make sub_category_id NOT NULL again in recurrent_movements table
ALTER TABLE recurrent_movements
    ALTER COLUMN sub_category_id SET NOT NULL; 