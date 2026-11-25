-- Dedup do import (FindExistingHashes): WHERE user_id = ? AND idempotency_hash IN (...).
-- A migration 019 criou a coluna e o seu .down já dropava este índice, mas o .up nunca
-- chegou a criá-lo. Parcial porque só movements vindos de import têm hash; os criados
-- manualmente ficam com NULL e não precisam entrar no índice.
CREATE INDEX IF NOT EXISTS idx_movements_idempotency_hash
    ON movements (user_id, idempotency_hash)
    WHERE idempotency_hash IS NOT NULL;

-- Série de parcelas (FindByInstallmentGroupFromNumber):
-- WHERE installment_group_id = ? AND installment_number >= ?, ordenado por installment_number.
-- Colunas na ordem da query para que o índice sirva ao filtro e à ordenação.
CREATE INDEX IF NOT EXISTS idx_movements_installment_group
    ON movements (installment_group_id, installment_number)
    WHERE installment_group_id IS NOT NULL;
