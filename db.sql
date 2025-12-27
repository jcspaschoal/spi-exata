/*
 * ==============================================================================================
 * CORE INFRASTRUCTURE - ENTERPRISE MULTI-TENANT SAAS (NORMALIZED)
 * ==============================================================================================
 * Engine: PostgreSQL 18
 * Revision: 2.1 (Widget Sizing Nullable)
 * ==============================================================================================
 */

BEGIN;

-- 1. CONFIGURAÇÕES
SET client_min_messages TO WARNING;
CREATE SCHEMA IF NOT EXISTS "public";

-- 2. ENUMS E DOMÍNIOS
DO $$ BEGIN
    CREATE TYPE "actions_enum" AS ENUM ('create', 'delete', 'get', 'update', 'publish');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- 3. TENANCY (Raiz)
CREATE TABLE "public"."tenant" (
                                   "tenant_id"   uuid NOT NULL DEFAULT uuidv7(),
                                   "name"        varchar(256) NOT NULL,
                                   "slug"        varchar(64) NOT NULL,
                                   "enabled"     boolean NOT NULL DEFAULT true,
                                   "created_at"  timestamptz NOT NULL DEFAULT now(),
                                   "updated_at"  timestamptz NOT NULL DEFAULT now(),

                                   CONSTRAINT "pk_tenant" PRIMARY KEY ("tenant_id"),
                                   CONSTRAINT "uq_tenant_slug" UNIQUE ("slug")
);
CREATE INDEX "idx_tenant_slug" ON "public"."tenant" ("slug");

-- 4. METADADOS GLOBAIS
CREATE TABLE "public"."role" (
                                 "role_id" smallint NOT NULL,
                                 "name"    varchar(50) NOT NULL,
                                 CONSTRAINT "pk_role" PRIMARY KEY ("role_id"),
                                 CONSTRAINT "uq_role_name" UNIQUE ("name")
);

CREATE TABLE "public"."resource_type" (
                                          "resource_type_id" smallint NOT NULL,
                                          "name"             varchar(50) NOT NULL,
                                          CONSTRAINT "pk_resource_type" PRIMARY KEY ("resource_type_id"),
                                          CONSTRAINT "uq_resource_type_composite" UNIQUE ("resource_type_id", "name")
);

CREATE TABLE "public"."layout" (
                                   "layout_id" smallint NOT NULL PRIMARY KEY,
                                   "name" varchar NOT NULL UNIQUE
);

CREATE TABLE "public"."feed_category" (
                                          "feed_category_id" smallint NOT NULL PRIMARY KEY,
                                          "name" varchar NOT NULL UNIQUE
);

-- [[ ALTERAÇÃO AQUI: 'size' agora permite NULL ]] --
CREATE TABLE "public"."widget_type" (
                                        "widget_type_id" smallint NOT NULL PRIMARY KEY,
                                        "name" varchar NOT NULL UNIQUE,
                                        "size" varchar(32) -- NULLABLE: Front-end deve decidir o default se vier null
);

-- 5. USUÁRIOS (Identidade Pura)
CREATE TABLE "public"."users" (
                                  "user_id"    uuid NOT NULL DEFAULT uuidv7(),
                                  "role_id"    smallint NOT NULL,
                                  "name"       varchar(256) NOT NULL,
                                  "email"      varchar(120) NOT NULL,
                                  "phone"      varchar(16),
                                  "password"   char(60) NOT NULL,
                                  "enabled"    boolean NOT NULL DEFAULT true,
                                  "created_at" timestamptz NOT NULL DEFAULT now(),
                                  "updated_at" timestamptz NOT NULL DEFAULT now(),

                                  CONSTRAINT "pk_users" PRIMARY KEY ("user_id"),
                                  CONSTRAINT "uq_users_email" UNIQUE ("email"),
                                  CONSTRAINT "fk_users_role" FOREIGN KEY ("role_id") REFERENCES "public"."role"("role_id")
);

CREATE TABLE "public"."password_reset_token" (
                                                 "user_id"    uuid NOT NULL,
                                                 "token_hash" varchar NOT NULL,
                                                 "expires_at" timestamptz NOT NULL,
                                                 CONSTRAINT "pk_password_reset" PRIMARY KEY ("user_id"),
                                                 CONSTRAINT "fk_token_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("user_id") ON DELETE CASCADE
);

-- 6. VÍNCULO DE TENANCY (Membership 1:1)
CREATE TABLE "public"."tenant_membership" (
                                              "user_id"    uuid NOT NULL,
                                              "tenant_id"  uuid NOT NULL,
                                              "created_at" timestamptz NOT NULL DEFAULT now(),

                                              CONSTRAINT "pk_tenant_membership" PRIMARY KEY ("user_id"),
                                              CONSTRAINT "fk_membership_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("user_id") ON DELETE CASCADE,
                                              CONSTRAINT "fk_membership_tenant" FOREIGN KEY ("tenant_id") REFERENCES "public"."tenant"("tenant_id") ON DELETE CASCADE
);
CREATE INDEX "idx_membership_tenant" ON "public"."tenant_membership" ("tenant_id");


-- 7. REGISTRY (Supertype Normalizado)
CREATE TABLE "public"."resource" (
                                     "resource_id"      uuid NOT NULL DEFAULT uuidv7(),
                                     "resource_type_id" smallint NOT NULL,

                                     CONSTRAINT "pk_resource" PRIMARY KEY ("resource_id"),
                                     CONSTRAINT "fk_resource_type" FOREIGN KEY ("resource_type_id") REFERENCES "public"."resource_type"("resource_type_id"),
                                     CONSTRAINT "uq_resource_integrity" UNIQUE ("resource_id", "resource_type_id")
);

-- 8. DASHBOARD (Âncora de Tenancy)
CREATE TABLE "public"."dashboard" (
                                      "dashboard_id"     uuid NOT NULL,
                                      "resource_type_id" smallint GENERATED ALWAYS AS (1) STORED,
                                      "tenant_id"        uuid NOT NULL,
                                      "name"             varchar NOT NULL,
                                      "domain"           varchar(255),
                                      "logo"             bytea,
                                      "created_at"       timestamptz NOT NULL DEFAULT now(),
                                      "updated_at"       timestamptz NOT NULL DEFAULT now(),

                                      CONSTRAINT "pk_dashboard" PRIMARY KEY ("dashboard_id"),
                                      CONSTRAINT "uq_dashboard_domain" UNIQUE ("domain"),

                                      CONSTRAINT "fk_dashboard_resource_integrity" FOREIGN KEY ("dashboard_id", "resource_type_id")
                                          REFERENCES "public"."resource"("resource_id", "resource_type_id") ON DELETE CASCADE,

                                      CONSTRAINT "fk_dashboard_tenant" FOREIGN KEY ("tenant_id") REFERENCES "public"."tenant"("tenant_id") ON DELETE RESTRICT
);
CREATE INDEX "idx_dashboard_domain" ON "public"."dashboard" ("domain");
CREATE INDEX "idx_dashboard_tenant" ON "public"."dashboard" ("tenant_id");

-- 9. CONTROLE DE ACESSO
CREATE TABLE "public"."user_dashboard_access" (
                                                  "user_id"      uuid NOT NULL,
                                                  "dashboard_id" uuid NOT NULL,
                                                  "tenant_id"    uuid NOT NULL,
                                                  "created_at"   timestamptz NOT NULL DEFAULT now(),

                                                  CONSTRAINT "pk_user_dashboard_access" PRIMARY KEY ("user_id", "dashboard_id"),
                                                  CONSTRAINT "fk_access_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("user_id") ON DELETE CASCADE,
                                                  CONSTRAINT "fk_access_dashboard" FOREIGN KEY ("dashboard_id") REFERENCES "public"."dashboard"("dashboard_id") ON DELETE CASCADE
);
CREATE INDEX "idx_access_dashboard_reverse" ON "public"."user_dashboard_access" ("dashboard_id");


-- 10. DEMAIS ENTIDADES (Page, Feed, Subject)

CREATE TABLE "public"."feed" (
                                 "feed_id"     uuid NOT NULL DEFAULT uuidv7(),
                                 "keywords"    varchar(500) NOT NULL,
                                 "category_id" smallint NOT NULL,

                                 CONSTRAINT "pk_feed" PRIMARY KEY ("feed_id"),
                                 CONSTRAINT "fk_feed_category" FOREIGN KEY ("category_id") REFERENCES "public"."feed_category"("feed_category_id")
);

CREATE TABLE "public"."page" (
                                 "page_id"          uuid NOT NULL,
                                 "resource_type_id" smallint GENERATED ALWAYS AS (2) STORED,
                                 "dashboard_id"     uuid NOT NULL,
                                 "layout_id"        smallint NOT NULL,
                                 "title"            varchar NOT NULL,
                                 "text"             text,
                                 "order"            smallint,
                                 "feed_id"          uuid,
                                 "created_at"       timestamptz NOT NULL DEFAULT now(),
                                 "updated_at"       timestamptz NOT NULL DEFAULT now(),

                                 CONSTRAINT "pk_page" PRIMARY KEY ("page_id"),

                                 CONSTRAINT "fk_page_resource_integrity" FOREIGN KEY ("page_id", "resource_type_id")
                                     REFERENCES "public"."resource"("resource_id", "resource_type_id") ON DELETE CASCADE,

                                 CONSTRAINT "fk_page_dashboard" FOREIGN KEY ("dashboard_id") REFERENCES "public"."dashboard"("dashboard_id") ON DELETE CASCADE,
                                 CONSTRAINT "fk_page_layout" FOREIGN KEY ("layout_id") REFERENCES "public"."layout"("layout_id"),
                                 CONSTRAINT "fk_page_feed" FOREIGN KEY ("feed_id") REFERENCES "public"."feed"("feed_id")
);
CREATE INDEX "idx_page_dashboard_order" ON "public"."page" ("dashboard_id", "order");

CREATE TABLE "public"."subject" (
                                    "subject_id"           uuid NOT NULL,
                                    "resource_type_id"     smallint GENERATED ALWAYS AS (3) STORED,
                                    "page_id"              uuid,
                                    "widget_id"            smallint NOT NULL,
                                    "title"                varchar NOT NULL,
                                    "order"                smallint NOT NULL,
                                    "description"          varchar NOT NULL,
                                    "result"               jsonb NOT NULL,
                                    "analyst_modification" jsonb,
                                    "created_at"           timestamptz NOT NULL DEFAULT now(),
                                    "updated_at"           timestamptz NOT NULL DEFAULT now(),

                                    CONSTRAINT "pk_subject" PRIMARY KEY ("subject_id"),

                                    CONSTRAINT "fk_subject_resource_integrity" FOREIGN KEY ("subject_id", "resource_type_id")
                                        REFERENCES "public"."resource"("resource_id", "resource_type_id") ON DELETE CASCADE,

                                    CONSTRAINT "fk_subject_page" FOREIGN KEY ("page_id") REFERENCES "public"."page"("page_id") ON DELETE CASCADE,
                                    CONSTRAINT "fk_subject_widget" FOREIGN KEY ("widget_id") REFERENCES "public"."widget_type"("widget_type_id")
);

ALTER TABLE "public"."subject" ALTER COLUMN "result" SET STORAGE EXTERNAL;
CREATE INDEX "idx_subject_page_order" ON "public"."subject" ("page_id", "order");
CREATE INDEX "idx_subject_result_gin" ON "public"."subject" USING GIN ("result" jsonb_path_ops);

COMMIT;