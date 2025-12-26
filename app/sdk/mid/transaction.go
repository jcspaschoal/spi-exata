package mid

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/jcpaschoal/spi-exata/app/sdk/errs"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
	"github.com/jcpaschoal/spi-exata/foundatiton/logger"
)

// BeginCommitRollback starts a transaction for the domain call.
func BeginCommitRollback(log *logger.Logger, bgn sqldb.Beginner) web.MidFunc {
	m := func(next web.HandlerFunc) web.HandlerFunc {
		h := func(ctx context.Context, r *http.Request) web.Encoder {
			hasCommitted := false

			log.Info(ctx, "BEGIN TRANSACTION")
			// 1. Inicia a Transação física no banco
			tx, err := bgn.Begin()
			if err != nil {
				return errs.Errorf(errs.Internal, "BEGIN TRANSACTION: %s", err)
			}

			// 2. Garante o Rollback em caso de pânico ou erro não tratado
			defer func() {
				if !hasCommitted {
					log.Info(ctx, "ROLLBACK TRANSACTION")
				}

				if err := tx.Rollback(); err != nil {
					if errors.Is(err, sql.ErrTxDone) {
						return
					}
					log.Info(ctx, "ROLLBACK TRANSACTION", "ERROR", err)
				}
			}()

			// 4. Injeta a transação no contexto
			ctx = setTran(ctx, tx)

			// 5. Passa para o próximo Handler (Business Logic)
			resp := next(ctx, r)

			if checkIsError(resp) != nil {
				return resp
			}

			// 6. Efetiva a transação
			log.Info(ctx, "COMMIT TRANSACTION")
			if err := tx.Commit(); err != nil {
				return errs.Errorf(errs.Internal, "COMMIT TRANSACTION: %s", err)
			}

			hasCommitted = true

			return resp
		}

		return h
	}

	return m
}
