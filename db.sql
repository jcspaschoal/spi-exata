-- ==========================================
-- 1. EXTENSÕES, TIPOS E SEGURANÇA
-- ==========================================
CREATE SCHEMA IF NOT EXISTS "public";

CREATE TYPE "actions_enum" AS ENUM ('create', 'delete', 'get', 'update');

-- ==========================================
-- 2. TABELAS DE DOMÍNIO (LOOKUP)
-- ==========================================

CREATE TABLE "public"."role" (
                                 "role_id" smallint NOT NULL,
                                 "name" varchar(50) NOT NULL,
                                 CONSTRAINT "pk_role" PRIMARY KEY ("role_id"),
                                 CONSTRAINT "uq_role_name" UNIQUE ("name")
);

CREATE TABLE "public"."resource_type" (
                                          "resource_type_id" smallint NOT NULL,
                                          "name" varchar(50) NOT NULL,
                                          CONSTRAINT "pk_resource_type" PRIMARY KEY ("resource_type_id"),
                                          CONSTRAINT "uq_resource_type_name" UNIQUE ("name"),
                                          CONSTRAINT "uq_resource_type_composite" UNIQUE ("resource_type_id", "name")
);

-- ==========================================
-- 3. REGISTRY POLIMÓRFICO (Garantia de Consistência)
-- ==========================================

CREATE TABLE "public"."resource" (
                                     "resource_id" uuid NOT NULL,
                                     "resource_type_id" smallint NOT NULL,
                                     CONSTRAINT "pk_resource" PRIMARY KEY ("resource_id"),
                                     CONSTRAINT "fk_resource_type" FOREIGN KEY ("resource_type_id")
                                         REFERENCES "public"."resource_type"("resource_type_id"),
                                     CONSTRAINT "uq_resource_composite" UNIQUE ("resource_id", "resource_type_id")
);

-- ==========================================
-- 4. GESTÃO DE USUÁRIOS
-- ==========================================

CREATE TABLE "public"."user" (
                                 "user_id" uuid NOT NULL,
                                 "role_id" smallint NOT NULL,
                                 "name" varchar(256) NOT NULL,
                                 "email" varchar(120) NOT NULL,
                                 "password" char(60) NOT NULL, -- Otimizado para Bcrypt
                                 "phone" varchar(16),
                                 "created_at" timestamptz NOT NULL DEFAULT now(),
                                 "updated_at" timestamptz NOT NULL DEFAULT now(),
                                 "enabled" boolean NOT NULL DEFAULT true,
                                 CONSTRAINT "pk_user" PRIMARY KEY ("user_id"),
                                 CONSTRAINT "uq_user_email" UNIQUE ("email"),
                                 CONSTRAINT "uq_user_phone" UNIQUE ("phone"),
                                 CONSTRAINT "fk_user_role" FOREIGN KEY ("role_id") REFERENCES "public"."role"("role_id")
);

CREATE TABLE "public"."password_reset_token" (
                                                 "user_id" uuid NOT NULL,
                                                 "token_hash" varchar NOT NULL,
                                                 "expires_at" timestamptz NOT NULL,
                                                 "created_at" timestamptz NOT NULL DEFAULT now(),
                                                 CONSTRAINT "pk_password_reset_token" PRIMARY KEY ("user_id"),
                                                 CONSTRAINT "fk_password_reset_token_user" FOREIGN KEY ("user_id") REFERENCES "public"."user"("user_id")
);

-- ==========================================
-- 5. RECURSOS (Entidades de Negócio)
-- ==========================================

-- DASHBOARD (Type ID: 1)
CREATE TABLE "public"."dashboard" (
                                      "dashboard_id" uuid NOT NULL,
                                      "resource_type_id" smallint GENERATED ALWAYS AS (1) STORED,
                                      "name" varchar NOT NULL,
                                      "logo" bytea,
                                      CONSTRAINT "pk_dashboard" PRIMARY KEY ("dashboard_id"),
                                      CONSTRAINT "uq_dashboard_name" UNIQUE ("name"),
                                      CONSTRAINT "fk_dashboard_resource" FOREIGN KEY ("dashboard_id", "resource_type_id")
                                          REFERENCES "public"."resource"("resource_id", "resource_type_id") ON DELETE CASCADE
);

-- PAGE (Type ID: 2)
CREATE TABLE "public"."layout" (
                                   "layout_id" smallint NOT NULL,
                                   "name" varchar NOT NULL,
                                   "description" varchar NOT NULL,
                                   CONSTRAINT "pk_layout" PRIMARY KEY ("layout_id"),
                                   CONSTRAINT "uq_layout_name" UNIQUE ("name")
);

CREATE TABLE "public"."feed_category" (
                                          "feed_category_id" smallint NOT NULL,
                                          "name" varchar NOT NULL,
                                          CONSTRAINT "pk_feed_category" PRIMARY KEY ("feed_category_id"),
                                          CONSTRAINT "uq_feed_category_name" UNIQUE ("name")
);

CREATE TABLE "public"."feed" (
                                 "feed_id" uuid NOT NULL,
                                 "keywords" varchar(500) NOT NULL,
                                 "category_id" smallint NOT NULL,
                                 CONSTRAINT "pk_feed" PRIMARY KEY ("feed_id"),
                                 CONSTRAINT "fk_feed_feed_category" FOREIGN KEY ("category_id") REFERENCES "public"."feed_category"("feed_category_id")
);

CREATE TABLE "public"."page" (
                                 "page_id" uuid NOT NULL,
                                 "resource_type_id" smallint GENERATED ALWAYS AS (2) STORED,
                                 "dashboard_id" uuid NOT NULL,
                                 "layout_id" smallint NOT NULL,
                                 "title" varchar NOT NULL,
                                 "text" text,
                                 "order" smallint,
                                 "feed_id" uuid,
                                 CONSTRAINT "pk_page" PRIMARY KEY ("page_id"),
                                 CONSTRAINT "fk_page_resource" FOREIGN KEY ("page_id", "resource_type_id")
                                     REFERENCES "public"."resource"("resource_id", "resource_type_id") ON DELETE CASCADE,
                                 CONSTRAINT "fk_page_dashboard" FOREIGN KEY ("dashboard_id") REFERENCES "public"."dashboard"("dashboard_id"),
                                 CONSTRAINT "fk_page_layout" FOREIGN KEY ("layout_id") REFERENCES "public"."layout"("layout_id"),
                                 CONSTRAINT "fk_page_feed" FOREIGN KEY ("feed_id") REFERENCES "public"."feed"("feed_id")
);

-- SUBJECT (Type ID: 3)
CREATE TABLE "public"."widget_type" (
                                        "widget_type_id" smallint NOT NULL,
                                        "name" varchar NOT NULL,
                                        CONSTRAINT "pk_widget_type" PRIMARY KEY ("widget_type_id"),
                                        CONSTRAINT "uq_widget_type_name" UNIQUE ("name")
);

CREATE TABLE "public"."subject" (
                                    "subject_id" uuid NOT NULL,
                                    "resource_type_id" smallint GENERATED ALWAYS AS (3) STORED,
                                    "title" varchar NOT NULL,
                                    "order" smallint NOT NULL,
                                    "description" varchar NOT NULL,
                                    "analyst_modification" jsonb,
                                    "result" jsonb NOT NULL,
                                    "widget_id" smallint NOT NULL,
                                    "created_at" timestamptz NOT NULL DEFAULT now(),
                                    "updated_at" timestamptz NOT NULL DEFAULT now(),
                                    "page_id" uuid,
                                    CONSTRAINT "pk_subject" PRIMARY KEY ("subject_id"),
                                    CONSTRAINT "fk_subject_resource" FOREIGN KEY ("subject_id", "resource_type_id")
                                        REFERENCES "public"."resource"("resource_id", "resource_type_id") ON DELETE CASCADE,
                                    CONSTRAINT "fk_subject_page" FOREIGN KEY ("page_id") REFERENCES "public"."page"("page_id"),
                                    CONSTRAINT "fk_subject_widget_type" FOREIGN KEY ("widget_id") REFERENCES "public"."widget_type"("widget_type_id")
);

-- ==========================================
-- 6. ACL (Access Control List)
-- ==========================================

CREATE TABLE "public"."acl" (
                                "resource_id" uuid NOT NULL,
                                "user_id" uuid NOT NULL,
                                "resource_type_id" smallint NOT NULL,
                                "actions" actions_enum NOT NULL,
                                CONSTRAINT "pk_acl" PRIMARY KEY ("resource_id", "user_id"),
                                CONSTRAINT "fk_acl_resource_registry" FOREIGN KEY ("resource_id", "resource_type_id")
                                    REFERENCES "public"."resource"("resource_id", "resource_type_id"),
                                CONSTRAINT "fk_acl_user" FOREIGN KEY ("user_id") REFERENCES "public"."user"("user_id")
);

-- ==========================================
-- 7. TUNING DE STORAGE (JSONB & BYTEA)
-- ==========================================

-- Reduz overhead de CPU na descompressão para buscas intensivas
ALTER TABLE "public"."subject" ALTER COLUMN "result" SET STORAGE EXTERNAL;
ALTER TABLE "public"."subject" ALTER COLUMN "analyst_modification" SET STORAGE EXTERNAL;


-- ==========================================
-- 8. ÍNDICES E PERFORMANCE (POSTGRES 18)
-- ==========================================

-- Índice ACL otimizado para Index-Only Scan (Cobre 90% das queries de permissão)
CREATE INDEX "idx_acl_lookup_composite" ON "public"."acl" ("user_id", "resource_type_id")
    INCLUDE ("actions", "resource_id");

-- GIN com path_ops para buscas ultra-rápidas dentro do JSONB (Containment)
CREATE INDEX "idx_subject_result_path_ops" ON "public"."subject" USING GIN ("result" jsonb_path_ops);
CREATE INDEX "idx_subject_mod_path_ops" ON "public"."subject" USING GIN ("analyst_modification" jsonb_path_ops);

-- B-Tree para ordenação e filtragem de hierarquia
CREATE INDEX "idx_page_dash_hierarchy" ON "public"."page" ("dashboard_id", "order", "page_id");
CREATE INDEX "idx_subject_page_hierarchy" ON "public"."subject" ("page_id", "order", "subject_id");

-- Índice para busca rápida de recursos por tipo no Registry
CREATE INDEX "idx_resource_registry_lookup" ON "public"."resource" ("resource_type_id", "resource_id");