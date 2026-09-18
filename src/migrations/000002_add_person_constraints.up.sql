ALTER TABLE persons
    ADD CONSTRAINT persons_name_not_blank CHECK (btrim(name) <> ''),
    ADD CONSTRAINT persons_age_non_negative CHECK (age IS NULL OR age >= 0);
