-- Delete all subcategories first
DELETE FROM sub_categories WHERE user_id = 'default_category_id';

-- Then delete all categories
DELETE FROM categories WHERE user_id = 'default_category_id'; 