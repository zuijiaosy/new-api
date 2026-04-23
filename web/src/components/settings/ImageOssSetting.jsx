import React, { useEffect, useState } from 'react';
import {
  Banner,
  Button,
  Divider,
  Form,
  Space,
  Spin,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const SECTION_TITLE_STYLE = { marginTop: 20 };

const DEFAULT_VALUES = {
  enabled: false,
  fallback_to_upstream: false,
  endpoint: '',
  access_key: '',
  secret_key: '',
  bucket: 'new-api-images',
  region: 'us-east-1',
  use_ssl: false,
  use_path_style: true,
  public_url_prefix: '',
  retention_hours: 24,
  download_timeout_seconds: 30,
  cleanup_interval_hours: 24,
  cleanup_batch_size: 500,
};

const ImageOssSetting = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [pinging, setPinging] = useState(false);
  const [cleaning, setCleaning] = useState(false);
  const [values, setValues] = useState(DEFAULT_VALUES);

  const loadConfig = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/option/');
      const { success, data } = res.data || {};
      if (!success) return;
      const raw = data.find((i) => i.key === 'oss_image_setting');
      if (!raw) return;
      let parsed = {};
      try {
        parsed = JSON.parse(raw.value);
      } catch (_) {
        // 忽略解析失败，沿用默认值
      }
      setValues({ ...DEFAULT_VALUES, ...parsed });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadConfig();
  }, []);

  const handleSave = async () => {
    setSaving(true);
    try {
      // 脱敏占位（**** 开头）等价于“不修改已有 SecretKey”，提交前置空
      const payload = { ...values };
      if (
        typeof payload.secret_key === 'string' &&
        payload.secret_key.startsWith('****')
      ) {
        payload.secret_key = '';
      }
      const res = await API.put('/api/option/', {
        key: 'oss_image_setting',
        value: JSON.stringify(payload),
      });
      if (res.data.success) {
        showSuccess(t('保存成功'));
        loadConfig();
      } else {
        showError(res.data.message);
      }
    } finally {
      setSaving(false);
    }
  };

  const handlePing = async () => {
    setPinging(true);
    try {
      const res = await API.post('/api/oss/images/ping', values);
      if (res.data.success) {
        Toast.success(`${t('连接成功')} (${res.data.latency_ms}ms)`);
      } else {
        Toast.error(`${t('连接失败')}: ${res.data.message}`);
      }
    } finally {
      setPinging(false);
    }
  };

  const handleCleanup = async () => {
    setCleaning(true);
    try {
      const res = await API.post('/api/oss/images/cleanup');
      if (res.data.success) {
        Toast.success(
          `${t('清理完成')}: ${t('扫描')} ${res.data.scanned} / ${t('删除')} ${res.data.deleted} / ${t('失败')} ${res.data.failed}`,
        );
      } else {
        Toast.error(res.data.message);
      }
    } finally {
      setCleaning(false);
    }
  };

  if (loading) return <Spin />;

  return (
    <div style={{ padding: 16 }}>
      {!values.enabled && (
        <Banner
          type='info'
          description={t('当前功能未启用，开启开关后配置才会生效')}
          style={{ marginBottom: 16 }}
        />
      )}

      <Form
        initValues={values}
        onValueChange={(v) => setValues({ ...values, ...v })}
      >
        <Typography.Title heading={5}>{t('基础设置')}</Typography.Title>
        <Form.Switch field='enabled' label={t('启用图片 OSS 转存')} />
        <Form.Switch
          field='fallback_to_upstream'
          label={t('失败时回退到上游 URL')}
        />

        <Typography.Title heading={5} style={SECTION_TITLE_STYLE}>
          {t('MinIO 连接')}
        </Typography.Title>
        <Form.Input
          field='endpoint'
          label='Endpoint'
          placeholder='127.0.0.1:9000'
        />
        <Form.Input field='access_key' label='AccessKey' />
        <Form.Input field='secret_key' label='SecretKey' mode='password' />
        <Form.Input field='bucket' label='Bucket' />
        <Form.Input field='region' label='Region' />
        <Form.Switch field='use_ssl' label='Use SSL' />
        <Form.Switch field='use_path_style' label='Path Style' />
        <Form.Input
          field='public_url_prefix'
          label={t('公网 URL 前缀')}
          placeholder='https://cdn.example.com'
        />
        <Space>
          <Button onClick={handlePing} loading={pinging}>
            {t('连接测试')}
          </Button>
        </Space>

        <Typography.Title heading={5} style={SECTION_TITLE_STYLE}>
          {t('生命周期')}
        </Typography.Title>
        <Form.InputNumber
          field='retention_hours'
          label={t('保留小时数')}
          min={1}
        />
        <Form.InputNumber
          field='download_timeout_seconds'
          label={t('下载超时(秒)')}
          min={1}
        />
        <Form.InputNumber
          field='cleanup_interval_hours'
          label={t('清理周期(小时)')}
          min={1}
          extraText={t('修改后下次进程启动生效；如需立即生效请使用下方手动清理按钮')}
        />
        <Form.InputNumber
          field='cleanup_batch_size'
          label={t('清理批大小')}
          min={1}
        />
        <Space>
          <Button onClick={handleCleanup} loading={cleaning}>
            {t('立即执行清理')}
          </Button>
        </Space>
      </Form>

      <Divider style={{ margin: '20px 0' }} />
      <Space>
        <Button
          theme='solid'
          type='primary'
          loading={saving}
          onClick={handleSave}
        >
          {t('保存配置')}
        </Button>
      </Space>
    </div>
  );
};

export default ImageOssSetting;
