# Stripe 구독 크레딧 시스템용 PostgreSQL 데이터베이스 스키마

## 핵심 아키텍처 개요

이 데이터베이스 스키마는 사용자가 정기 결제가 성공하면 크레딧을 받고 서비스를 사용하는 동안 해당 크레딧을 소비하는 구독 서비스를 처리합니다. **이 시스템은 성능과 안정성을 위해 로컬 결제 데이터를 유지하면서 외부 Supabase 인증과 통합됩니다.** 주요 기능으로는 포괄적인 Stripe 웹훅 처리, 경쟁 조건 방지 및 강력한 감사 추적이 있습니다.

## 완전한 데이터베이스 스키마

### 1. Extensions and Types

```sql
-- Required PostgreSQL extensions
-- Required PostgreSQL extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 구독 상태 (심플 버전)
CREATE TYPE subscription_status AS ENUM (
    'active',      -- 활성 (정상 구독 중)
    'inactive'     -- 비활성화됨
);

-- 크레딧 거래 타입 (심플 버전)
CREATE TYPE transaction_type AS ENUM (
    'credit_allocation',  -- 크레딧 할당 (결제 시 크레딧 지급)
    'credit_usage',       -- 크레딧 사용
    'refund',            -- 환불
    'adjustment'         -- 조정 (관리자 수동 조정)
);

-- 웹훅 처리 상태
CREATE TYPE webhook_status AS ENUM (
    'pending',     -- 대기중
    'processing',  -- 처리중
    'completed',   -- 완료
    'failed'       -- 실패
);
```

### 2. Core table

```jsx
-- 구독 플랜 (최소한의 정보만)
CREATE TABLE subscription_plans (
    id BIGSERIAL PRIMARY KEY,
    stripe_price_id VARCHAR(100) UNIQUE NOT NULL,
    stripe_product_id VARCHAR(100) NOT NULL,

    -- 표시 정보
    display_name VARCHAR(200) NOT NULL,
    credits_per_cycle INTEGER NOT NULL,

    -- 우리 서비스 고유 정보
    features JSONB DEFAULT '{}',
    sort_order INTEGER DEFAULT 0,

    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_plans_stripe_price ON subscription_plans(stripe_price_id);
CREATE INDEX idx_plans_active ON subscription_plans(is_active) WHERE is_active = TRUE;

-- 사용자 구독 정보 (user_profiles 대체)
CREATE TABLE subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,  -- Supabase user ID (외부 참조, FK 없음)

    -- Stripe 정보
    stripe_customer_id VARCHAR(100) NOT NULL,
    stripe_subscription_id VARCHAR(100) UNIQUE,

    -- 구독 정보
    plan_id BIGINT REFERENCES subscription_plans(id),
    status subscription_status NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,

    -- 취소 처리
    canceled_at TIMESTAMPTZ,

    -- 메타데이터
    stripe_subscription_data JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 사용자당 하나의 활성 구독만 허용
CREATE UNIQUE INDEX unique_active_subscription_per_user
ON subscriptions (user_id)
WHERE status = 'active';

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_stripe_customer ON subscriptions(stripe_customer_id);
CREATE INDEX idx_subscriptions_stripe_sub ON subscriptions(stripe_subscription_id);
```

### 3. Credit System

```sql
-- 크레딧 거래 내역
CREATE TABLE credit_transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,  -- 외부 참조 (FK 없음)
    subscription_id BIGINT REFERENCES subscriptions(id),

    -- Transaction details
    transaction_type transaction_type NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    balance_after DECIMAL(15,2) NOT NULL,

    -- Description and metadata
    description TEXT NOT NULL,
    feature_name VARCHAR(100),
    usage_metadata JSONB DEFAULT '{}'::JSONB,

    -- External references
    reference_id VARCHAR(200),
    idempotency_key UUID UNIQUE,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    CONSTRAINT non_zero_amount CHECK (amount != 0),
    CONSTRAINT positive_balance CHECK (balance_after >= 0)
);

-- 인덱스
CREATE INDEX idx_credit_transactions_user_created ON credit_transactions(user_id, created_at DESC);
CREATE INDEX idx_credit_transactions_reference ON credit_transactions(reference_id)
WHERE reference_id IS NOT NULL;

-- 사용자별 현재 잔액 (Materialized View)
CREATE MATERIALIZED VIEW user_credit_balances AS
SELECT DISTINCT ON (user_id)
    user_id,
    balance_after as current_balance,
    created_at as last_transaction_at
FROM credit_transactions
ORDER BY user_id, created_at DESC;

CREATE UNIQUE INDEX idx_user_credit_balances_user_id ON user_credit_balances(user_id);

-- 크레딧 추가 함수 (변경 없음)
CREATE OR REPLACE FUNCTION add_credits(
    p_user_id UUID,
    p_amount DECIMAL,
    p_transaction_type transaction_type,
    p_description TEXT,
    p_reference_id VARCHAR DEFAULT NULL,
    p_subscription_id BIGINT DEFAULT NULL,
    p_idempotency_key UUID DEFAULT gen_random_uuid()
) RETURNS BIGINT AS $$
DECLARE
    current_balance DECIMAL;
    new_balance DECIMAL;
    transaction_id BIGINT;
    lock_key BIGINT;
BEGIN
    -- Generate user-specific lock key
    lock_key := ('x' || substr(p_user_id::text, 1, 16))::bit(64)::bigint;

    -- Acquire advisory lock for this user
    IF NOT pg_try_advisory_lock(lock_key) THEN
        RAISE EXCEPTION 'Could not acquire lock for user %. Please try again.', p_user_id;
    END IF;

    -- Check for existing transaction with same idempotency key
    SELECT id INTO transaction_id
    FROM credit_transactions
    WHERE idempotency_key = p_idempotency_key;

    IF transaction_id IS NOT NULL THEN
        PERFORM pg_advisory_unlock(lock_key);
        RETURN transaction_id; -- Already processed
    END IF;

    -- Get current balance
    SELECT COALESCE(balance_after, 0) INTO current_balance
    FROM credit_transactions
    WHERE user_id = p_user_id
    ORDER BY created_at DESC
    LIMIT 1;

    -- Calculate new balance
    new_balance := current_balance + p_amount;

    IF new_balance < 0 THEN
        PERFORM pg_advisory_unlock(lock_key);
        RAISE EXCEPTION 'Insufficient credits. Current balance: %, Requested: %', current_balance, p_amount;
    END IF;

    -- Insert transaction
    INSERT INTO credit_transactions (
        user_id, subscription_id, transaction_type, amount, balance_after,
        description, reference_id, idempotency_key
    ) VALUES (
        p_user_id, p_subscription_id, p_transaction_type, p_amount, new_balance,
        p_description, p_reference_id, p_idempotency_key
    ) RETURNING id INTO transaction_id;

    -- Refresh materialized view
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_credit_balances;

    -- Release lock
    PERFORM pg_advisory_unlock(lock_key);

    RETURN transaction_id;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 크레딧 사용 함수 (변경 없음)
CREATE OR REPLACE FUNCTION use_credits(
    p_user_id UUID,
    p_amount DECIMAL,
    p_feature_name VARCHAR,
    p_description TEXT,
    p_usage_metadata JSONB DEFAULT '{}'::JSONB,
    p_idempotency_key UUID DEFAULT gen_random_uuid()
) RETURNS BIGINT AS $$
DECLARE
    transaction_id BIGINT;
BEGIN
    SELECT add_credits(
        p_user_id := p_user_id,
        p_amount := -ABS(p_amount),
        p_transaction_type := 'credit_usage',
        p_description := p_description,
        p_reference_id := NULL,
        p_subscription_id := NULL,
        p_idempotency_key := p_idempotency_key
    ) INTO transaction_id;

    UPDATE credit_transactions
    SET
        feature_name = p_feature_name,
        usage_metadata = p_usage_metadata
    WHERE id = transaction_id;

    RETURN transaction_id;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 잔액 조회 함수
CREATE OR REPLACE FUNCTION get_user_balance(p_user_id UUID)
RETURNS DECIMAL AS $$
DECLARE
    balance DECIMAL;
BEGIN
    SELECT current_balance INTO balance
    FROM user_credit_balances
    WHERE user_id = p_user_id;

    RETURN COALESCE(balance, 0);
END;
$$ LANGUAGE plpgsql;
```

### 4. webhook processing

```sql
-- Webhook 이벤트 테이블 (변경 없음)
CREATE TABLE stripe_webhook_events (
    id BIGSERIAL PRIMARY KEY,
    stripe_event_id VARCHAR(255) NOT NULL UNIQUE,
    event_type VARCHAR(100) NOT NULL,
    status webhook_status DEFAULT 'pending',
    processed_at TIMESTAMPTZ,
    data JSONB NOT NULL,
    api_version VARCHAR(20),
    processing_attempts INTEGER DEFAULT 0,
    last_error TEXT,
    next_retry_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    stripe_created_at TIMESTAMPTZ
);

CREATE INDEX idx_webhook_events_stripe_id ON stripe_webhook_events(stripe_event_id);
CREATE INDEX idx_webhook_events_type ON stripe_webhook_events(event_type);
CREATE INDEX idx_webhook_events_status ON stripe_webhook_events(status);
CREATE INDEX idx_webhook_events_unprocessed ON stripe_webhook_events(created_at)
WHERE status IN ('pending', 'failed');

-- 결제 기록 (user_profiles 참조 제거)
CREATE TABLE payments (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,  -- 외부 참조
    subscription_id BIGINT REFERENCES subscriptions(id),

    -- Stripe payment details
    stripe_payment_intent_id VARCHAR(100) UNIQUE,
    stripe_charge_id VARCHAR(100),
    stripe_invoice_id VARCHAR(100),

    -- Payment information
    amount_cents INTEGER NOT NULL,
    currency VARCHAR(3) DEFAULT 'KRW',
    status VARCHAR(50) NOT NULL,

    -- Credits allocated
    credits_allocated DECIMAL(15,2) DEFAULT 0,
    credits_allocated_at TIMESTAMPTZ,

    -- Metadata
    payment_method_type VARCHAR(50),
    failure_code VARCHAR(100),
    failure_message TEXT,
    stripe_payment_data JSONB,

    -- Timestamps
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    CONSTRAINT positive_amount CHECK (amount_cents > 0),
    CONSTRAINT positive_credits CHECK (credits_allocated >= 0)
);

CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_subscription_id ON payments(subscription_id);
CREATE INDEX idx_payments_stripe_payment_intent ON payments(stripe_payment_intent_id);

-- Webhook 처리 함수 (변경 없음)
CREATE OR REPLACE FUNCTION process_stripe_webhook(
    p_stripe_event_id VARCHAR,
    p_event_type VARCHAR,
    p_event_data JSONB,
    p_api_version VARCHAR DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    webhook_id BIGINT;
    already_processed INTEGER;
    stripe_created_at TIMESTAMPTZ;
BEGIN
    stripe_created_at := TO_TIMESTAMP((p_event_data->>'created')::INTEGER);

    INSERT INTO stripe_webhook_events (
        stripe_event_id, event_type, data, api_version, stripe_created_at
    ) VALUES (
        p_stripe_event_id, p_event_type, p_event_data, p_api_version, stripe_created_at
    )
    ON CONFLICT (stripe_event_id) DO NOTHING;

    GET DIAGNOSTICS already_processed = ROW_COUNT;

    IF already_processed = 0 THEN
        RETURN FALSE;
    END IF;

    SELECT id INTO webhook_id
    FROM stripe_webhook_events
    WHERE stripe_event_id = p_stripe_event_id;

    UPDATE stripe_webhook_events
    SET status = 'processing'
    WHERE id = webhook_id;

    BEGIN
        CASE p_event_type
            WHEN 'invoice.payment_succeeded' THEN
                PERFORM handle_successful_payment(p_event_data);
            WHEN 'customer.subscription.updated' THEN
                PERFORM handle_subscription_updated(p_event_data);
            WHEN 'customer.subscription.deleted' THEN
                PERFORM handle_subscription_canceled(p_event_data);
            ELSE
                RAISE NOTICE 'Unhandled webhook event type: %', p_event_type;
        END CASE;

        UPDATE stripe_webhook_events
        SET status = 'completed', processed_at = NOW()
        WHERE id = webhook_id;

        RETURN TRUE;

    EXCEPTION WHEN OTHERS THEN
        UPDATE stripe_webhook_events
        SET
            status = 'failed',
            processing_attempts = processing_attempts + 1,
            last_error = SQLERRM,
            next_retry_at = NOW() + INTERVAL '5 minutes' * POWER(2, processing_attempts)
        WHERE id = webhook_id;
        RAISE;
    END;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 결제 성공 처리 (수정됨 - user_profiles 제거)
CREATE OR REPLACE FUNCTION handle_successful_payment(event_data JSONB)
RETURNS VOID AS $$
DECLARE
    invoice_data JSONB;
    subscription_id_stripe VARCHAR;
    payment_intent_id VARCHAR;
    amount_paid INTEGER;
    v_user_id UUID;
    v_subscription_id BIGINT;
    v_credits_to_allocate INTEGER;
    v_plan_name VARCHAR;
BEGIN
    invoice_data := event_data->'data'->'object';
    subscription_id_stripe := invoice_data->>'subscription';
    payment_intent_id := invoice_data->>'payment_intent';
    amount_paid := (invoice_data->>'amount_paid')::INTEGER;

    -- 구독 정보와 플랜 정보 조회
    SELECT
        s.id, s.user_id, p.credits_per_cycle, p.display_name
    INTO
        v_subscription_id, v_user_id, v_credits_to_allocate, v_plan_name
    FROM subscriptions s
    JOIN subscription_plans p ON s.plan_id = p.id
    WHERE s.stripe_subscription_id = subscription_id_stripe;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Subscription not found for Stripe ID: %', subscription_id_stripe;
    END IF;

    -- 결제 기록
    INSERT INTO payments (
        user_id, subscription_id, stripe_payment_intent_id,
        stripe_invoice_id, amount_cents, currency, status,
        paid_at, stripe_payment_data
    ) VALUES (
        v_user_id, v_subscription_id, payment_intent_id,
        invoice_data->>'id', amount_paid, COALESCE(invoice_data->>'currency', 'krw'), 'succeeded',
        TO_TIMESTAMP((invoice_data->>'status_transitions'->>'paid_at')::INTEGER),
        event_data
    )
    ON CONFLICT (stripe_payment_intent_id) DO NOTHING;

    -- 크레딧 할당
    PERFORM add_credits(
        p_user_id := v_user_id,
        p_amount := v_credits_to_allocate,
        p_transaction_type := 'credit_allocation',
        p_description := format('%s 구독 결제', v_plan_name),
        p_reference_id := payment_intent_id,
        p_subscription_id := v_subscription_id,
        p_idempotency_key := gen_random_uuid()
    );

    -- 결제 기록 업데이트
    UPDATE payments
    SET
        credits_allocated = v_credits_to_allocate,
        credits_allocated_at = NOW()
    WHERE stripe_payment_intent_id = payment_intent_id;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 구독 업데이트 처리
CREATE OR REPLACE FUNCTION handle_subscription_updated(event_data JSONB)
RETURNS VOID AS $$
DECLARE
    sub_data JSONB;
    stripe_sub_id VARCHAR;
BEGIN
    sub_data := event_data->'data'->'object';
    stripe_sub_id := sub_data->>'id';

    UPDATE subscriptions
    SET
        status = CASE
            WHEN (sub_data->>'status')::TEXT = 'active' THEN 'active'::subscription_status
            ELSE 'inactive'::subscription_status
        END,
        current_period_start = TO_TIMESTAMP((sub_data->>'current_period_start')::INTEGER),
        current_period_end = TO_TIMESTAMP((sub_data->>'current_period_end')::INTEGER),
        canceled_at = CASE
            WHEN sub_data->>'canceled_at' IS NOT NULL
            THEN TO_TIMESTAMP((sub_data->>'canceled_at')::INTEGER)
            ELSE NULL
        END,
        stripe_subscription_data = sub_data,
        updated_at = NOW()
    WHERE stripe_subscription_id = stripe_sub_id;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 구독 취소 처리
CREATE OR REPLACE FUNCTION handle_subscription_canceled(event_data JSONB)
RETURNS VOID AS $$
DECLARE
    sub_data JSONB;
    stripe_sub_id VARCHAR;
BEGIN
    sub_data := event_data->'data'->'object';
    stripe_sub_id := sub_data->>'id';

    UPDATE subscriptions
    SET
        status = 'inactive'::subscription_status,
        canceled_at = NOW(),
        stripe_subscription_data = sub_data,
        updated_at = NOW()
    WHERE stripe_subscription_id = stripe_sub_id;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
```

### 5. Security and Audit Framework

```sql
-- 감사 로그 (user_profiles 참조 제거)
CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID,  -- 외부 참조
    action VARCHAR(100) NOT NULL,
    table_name VARCHAR(100) NOT NULL,
    record_id BIGINT,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    metadata JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX idx_audit_log_table_action ON audit_log(table_name, action);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);

-- 감사 트리거 함수
CREATE OR REPLACE FUNCTION audit_table_changes() RETURNS TRIGGER AS $$
DECLARE
    current_user_id UUID;
BEGIN
    current_user_id := COALESCE(
        (current_setting('app.current_user_id', true))::UUID,
        CASE
            WHEN TG_OP = 'DELETE' THEN OLD.user_id
            ELSE NEW.user_id
        END
    );

    IF TG_OP = 'DELETE' THEN
        INSERT INTO audit_log (user_id, action, table_name, record_id, old_values, ip_address)
        VALUES (current_user_id, 'DELETE', TG_TABLE_NAME, OLD.id, to_jsonb(OLD), inet_client_addr());
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO audit_log (user_id, action, table_name, record_id, old_values, new_values, ip_address)
        VALUES (current_user_id, 'UPDATE', TG_TABLE_NAME, NEW.id, to_jsonb(OLD), to_jsonb(NEW), inet_client_addr());
        RETURN NEW;
    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO audit_log (user_id, action, table_name, record_id, new_values, ip_address)
        VALUES (current_user_id, 'INSERT', TG_TABLE_NAME, NEW.id, to_jsonb(NEW), inet_client_addr());
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 주요 테이블에 감사 트리거 적용
CREATE TRIGGER audit_subscriptions
    AFTER INSERT OR UPDATE OR DELETE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION audit_table_changes();

CREATE TRIGGER audit_credit_transactions
    AFTER INSERT OR UPDATE OR DELETE ON credit_transactions
    FOR EACH ROW EXECUTE FUNCTION audit_table_changes();

CREATE TRIGGER audit_payments
    AFTER INSERT OR UPDATE OR DELETE ON payments
    FOR EACH ROW EXECUTE FUNCTION audit_table_changes();

-- RLS 설정 (간소화)
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE credit_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE payments ENABLE ROW LEVEL SECURITY;

-- 사용자는 자신의 데이터만 조회 가능
CREATE POLICY "Users can view own subscriptions"
ON subscriptions FOR SELECT
USING (user_id = (current_setting('app.current_user_id', true))::UUID);

CREATE POLICY "Users can view own credit transactions"
ON credit_transactions FOR SELECT
USING (user_id = (current_setting('app.current_user_id', true))::UUID);

CREATE POLICY "Users can view own payments"
ON payments FOR SELECT
USING (user_id = (current_setting('app.current_user_id', true))::UUID);

-- 서비스 역할은 모든 접근 가능 (애플리케이션용)
CREATE ROLE service_role;

GRANT ALL ON ALL TABLES IN SCHEMA public TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO service_role;
GRANT ALL ON ALL FUNCTIONS IN SCHEMA public TO service_role;
```

### 6. Performance Monitoring

```sql
-- 구독 분석
CREATE MATERIALIZED VIEW subscription_analytics AS
SELECT
    DATE_TRUNC('month', created_at) as month,
    status,
    COUNT(*) as subscription_count,
    COUNT(DISTINCT user_id) as unique_users,
    SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_count
FROM subscriptions
WHERE created_at >= CURRENT_DATE - INTERVAL '24 months'
GROUP BY DATE_TRUNC('month', created_at), status
ORDER BY month DESC, status;

CREATE INDEX idx_subscription_analytics_month_status ON subscription_analytics(month, status);

-- 크레딧 사용 분석
CREATE MATERIALIZED VIEW credit_usage_analytics AS
SELECT
    DATE_TRUNC('day', created_at) as date,
    feature_name,
    COUNT(*) as usage_count,
    COUNT(DISTINCT user_id) as unique_users,
    SUM(ABS(amount)) as total_credits_used,
    AVG(ABS(amount)) as avg_credits_per_use
FROM credit_transactions
WHERE transaction_type = 'credit_usage'
    AND created_at >= CURRENT_DATE - INTERVAL '90 days'
GROUP BY DATE_TRUNC('day', created_at), feature_name
ORDER BY date DESC, total_credits_used DESC;

-- 분석 뷰 갱신 함수
CREATE OR REPLACE FUNCTION refresh_analytics() RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY subscription_analytics;
    REFRESH MATERIALIZED VIEW CONCURRENTLY credit_usage_analytics;
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_credit_balances;
END;
$$ LANGUAGE plpgsql;

-- 시스템 상태 모니터링 (user_profiles 제거)
CREATE VIEW system_health AS
SELECT
    'total_subscriptions' as metric,
    COUNT(*)::TEXT as value,
    'count' as unit
FROM subscriptions
UNION ALL
SELECT
    'active_subscriptions',
    COUNT(*)::TEXT,
    'count'
FROM subscriptions WHERE status = 'active'
UNION ALL
SELECT
    'unique_users',
    COUNT(DISTINCT user_id)::TEXT,
    'count'
FROM subscriptions
UNION ALL
SELECT
    'pending_webhooks',
    COUNT(*)::TEXT,
    'count'
FROM stripe_webhook_events WHERE status = 'pending'
UNION ALL
SELECT
    'total_credit_balance',
    ROUND(SUM(current_balance), 2)::TEXT,
    'credits'
FROM user_credit_balances
UNION ALL
SELECT
    'failed_webhooks_24h',
    COUNT(*)::TEXT,
    'count'
FROM stripe_webhook_events
WHERE status = 'failed' AND created_at >= NOW() - INTERVAL '24 hours';
```

### 7. Helper functions

```jsx
-- 사용자 컨텍스트 설정 (RLS용)
CREATE OR REPLACE FUNCTION set_user_context(user_id UUID)
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_user_id', user_id::TEXT, true);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 사용자의 활성 구독 조회
CREATE OR REPLACE FUNCTION get_user_subscription(p_user_id UUID)
RETURNS TABLE (
    subscription_id BIGINT,
    plan_name VARCHAR,
    status subscription_status,
    current_period_end TIMESTAMPTZ,
    credits_remaining DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        s.id,
        p.display_name,
        s.status,
        s.current_period_end,
        COALESCE(b.current_balance, 0)
    FROM subscriptions s
    JOIN subscription_plans p ON s.plan_id = p.id
    LEFT JOIN user_credit_balances b ON s.user_id = b.user_id
    WHERE s.user_id = p_user_id
    AND s.status = 'active'
    ORDER BY s.created_at DESC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;
```

## Integration with Supabase Auth

### Setting Up User Context

```jsx
// Set current user context for RLS (in your application code)
const setUserContext = async (supabaseClient, userId) => {
  await supabaseClient.rpc('set_user_context', { user_id: userId });
};

-- SQL function to set user context
CREATE OR REPLACE FUNCTION set_user_context(user_id UUID)
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_user_id', user_id::TEXT, true);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

```
