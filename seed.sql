/*
 * ==============================================================================================
 * SEED DATA SCRIPT - GOVERNO SP & RJ
 * ==============================================================================================
 * 1. Insere Roles e Resource Types (Idempotente)
 * 2. Cria Tenant GOVERNO_SP e seu Dashboard (Domain: apexata.govsp.com)
 * 3. Cria Tenant GOVERNO_RJ e seu Dashboard (Domain: apexata.govrj.com)
 * ==============================================================================================
 */

BEGIN;

-- 1. POPULAR METADADOS (Roles e Resource Types)
-- Utilizamos ON CONFLICT para não quebrar se rodar o script duas vezes.

-- Roles
INSERT INTO "public"."role" ("role_id", "name") VALUES
                                                    (1, 'ADMIN'),
                                                    (2, 'USER'),
                                                    (3, 'ANALYST')
ON CONFLICT ("role_id") DO UPDATE SET "name" = EXCLUDED."name";

-- Resource Types
-- NOTA: Os IDs devem coincidir com as colunas GENERATED ALWAYS nas tabelas filhas do seu schema:
-- Dashboard = 1, Page = 2, Subject = 3
INSERT INTO "public"."resource_type" ("resource_type_id", "name") VALUES
                                                                      (1, 'DASHBOARD'),
                                                                      (2, 'PAGE'),
                                                                      (3, 'SUBJECT')
ON CONFLICT ("resource_type_id") DO UPDATE SET "name" = EXCLUDED."name";


-- 2. CRIAÇÃO DOS TENANTS E DASHBOARDS (Lógica Procedural)
DO $$
    DECLARE
        -- Variáveis para SP
        v_tenant_sp_id uuid;
        v_dash_sp_resource_id uuid;

        -- Variáveis para RJ
        v_tenant_rj_id uuid;
        v_dash_rj_resource_id uuid;
    BEGIN

        -- ========================================================================
        -- CASO 1: GOVERNO_SP
        -- ========================================================================

        -- A. Criar o Tenant SP
        INSERT INTO "public"."tenant" ("name", "slug", "enabled")
        VALUES ('GOVERNO_SP', 'governosp', true)
        RETURNING "tenant_id" INTO v_tenant_sp_id;

        -- B. Criar o Recurso Base (Resource) para o Dashboard SP
        -- Tipo 1 = DASHBOARD
        INSERT INTO "public"."resource" ("resource_type_id")
        VALUES (1)
        RETURNING "resource_id" INTO v_dash_sp_resource_id;

        -- C. Criar o Dashboard SP associado ao Tenant e ao Recurso
        -- O ID do Dashboard DEVE ser o mesmo do Resource criado acima
        INSERT INTO "public"."dashboard" (
            "dashboard_id",
            "tenant_id",
            "name",
            "domain"
        )
        VALUES (
                   v_dash_sp_resource_id, -- FK para Resource (Herança)
                   v_tenant_sp_id,        -- FK para Tenant (Anchor)
                   'GOVSP',
                   'apexata.govsp.com'
               );

        RAISE NOTICE 'Tenant GOVERNO_SP criado com ID: %', v_tenant_sp_id;
        RAISE NOTICE 'Dashboard GOVSP criado com Resource ID: %', v_dash_sp_resource_id;


        -- ========================================================================
        -- CASO 2: GOVERNO_RJ
        -- ========================================================================

        -- A. Criar o Tenant RJ
        INSERT INTO "public"."tenant" ("name", "slug", "enabled")
        VALUES ('GOVERNO_RJ', 'governorj', true)
        RETURNING "tenant_id" INTO v_tenant_rj_id;

        -- B. Criar o Recurso Base (Resource) para o Dashboard RJ
        INSERT INTO "public"."resource" ("resource_type_id")
        VALUES (1)
        RETURNING "resource_id" INTO v_dash_rj_resource_id;

        -- C. Criar o Dashboard RJ associado ao Tenant e ao Recurso
        INSERT INTO "public"."dashboard" (
            "dashboard_id",
            "tenant_id",spi
            "name",
            "domain"
        )
        VALUES (
                   v_dash_rj_resource_id,
                   v_tenant_rj_id,
                   'GOVRJ',
                   'apexata.govrj.com'
               );

        RAISE NOTICE 'Tenant GOVERNO_RJ criado com ID: %', v_tenant_rj_id;
        RAISE NOTICE 'Dashboard GOVRJ criado com Resource ID: %', v_dash_rj_resource_id;

    END $$;

COMMIT;