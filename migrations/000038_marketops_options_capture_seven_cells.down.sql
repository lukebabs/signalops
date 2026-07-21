DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM marketops_options_capture_sessions WHERE required_surface_cells > 5
  ) THEN
    RAISE EXCEPTION 'cannot restore five-cell capture constraint while seven-cell records exist';
  END IF;
END $$;

ALTER TABLE marketops_options_capture_sessions
  DROP CONSTRAINT IF EXISTS marketops_options_capture_sessions_required_surface_cells_check;

ALTER TABLE marketops_options_capture_sessions
  ADD CONSTRAINT marketops_options_capture_sessions_required_surface_cells_check
  CHECK (required_surface_cells BETWEEN 0 AND 5);
