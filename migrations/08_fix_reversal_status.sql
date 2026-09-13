-- Reversals must be VOIDED (audit only), never POSTED: the balance query
-- sums POSTED entries only, so a POSTED reversal subtracts the voided
-- original a second time and double-counts the void. Repair rows written
-- before the BuildReversal fix. Idempotent: re-runs match zero rows.
UPDATE journal_entries
SET status = 'VOIDED'
WHERE reversal_of IS NOT NULL
  AND status = 'POSTED';
