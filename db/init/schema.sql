-- Enum: salary_type
CREATE TYPE salary_type AS ENUM (
  'Hourly',
  'Daily',
  'Monthly',
  'Yearly'
);

-- Enum: currency
CREATE TYPE currency AS ENUM (
  'JPY',
  'USD'
  -- 必要があれば追加
);

-- Enum: job_type
CREATE TYPE job_type AS ENUM (
  'Unknown',
  'FullTime',
  'PartTime',
  'Contract',
  'Temporary',
  'Freelance',
  'Internship',
  'Other'
);

-- Job Postings Table
CREATE TABLE job_postings (
  id UUID PRIMARY KEY,
  title TEXT NOT NULL,
  company_name TEXT NOT NULL,
  prefecture_code TEXT NOT NULL,
  prefecture_name TEXT NOT NULL,
  municipality TEXT NOT NULL,
  summary_url TEXT NOT NULL,
  job_type job_type NOT NULL DEFAULT 'Unknown',
  salary_min_amount BIGINT NOT NULL,
  salary_max_amount BIGINT NOT NULL,
  salary_unit salary_type NOT NULL,
  salary_currency currency NOT NULL DEFAULT 'JPY',
  salary_is_fixed BOOLEAN NOT NULL DEFAULT false,
  posted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  job_name TEXT NOT NULL,
  holiday_policy TEXT NOT NULL,
  raise INTEGER,
  bonus INTEGER,
  description TEXT NOT NULL,
  requirements TEXT NOT NULL,
  holidays_per_year INTEGER,
  work_hours TEXT NOT NULL,
  benefits TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);