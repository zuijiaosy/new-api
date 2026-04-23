import React, { useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Col,
  Form,
  Row,
  Space,
  Spin,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  API,
  compareObjects,
  showError,
  showSuccess,
  showWarning,
  toBoolean,
} from '../../helpers';

const { Text } = Typography;

const SECRET_MASK_PREFIX = '****';

const DEFAULT_VALUES = {
  'oss_image_setting.enabled': false,
  'oss_image_setting.fallback_to_upstream': false,
  'oss_image_setting.endpoint': '',
  'oss_image_setting.access_key': '',
  'oss_image_setting.secret_key': '',
  'oss_image_setting.bucket': 'new-api-images',
  'oss_image_setting.region': 'us-east-1',
  'oss_image_setting.use_ssl': false,
  'oss_image_setting.use_path_style': true,
  'oss_image_setting.public_url_prefix': '',
  'oss_image_setting.retention_hours': 24,
  'oss_image_setting.download_timeout_seconds': 30,
  'oss_image_setting.cleanup_interval_hours': 24,
  'oss_image_setting.cleanup_batch_size': 500,
};

const ImageOssSetting = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [pinging, setPinging] = useState(false);
  const [cleaning, setCleaning] = useState(false);
  const [inputs, setInputs] = useState(DEFAULT_VALUES);
  const [inputsRow, setInputsRow] = useState(DEFAULT_VALUES);
  const refForm = useRef();

  const handleFieldChange = (fieldName) => (value) => {
    setInputs((prev) => ({ ...prev, [fieldName]: value }));
  };

  const loadConfig = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/option/');
      const { success, data } = res.data || {};
      if (!success) return;
      const current = { ...DEFAULT_VALUES };
      data.forEach((item) => {
        if (!(item.key in DEFAULT_VALUES)) return;
        if (typeof DEFAULT_VALUES[item.key] === 'boolean') {
          current[item.key] = toBoolean(item.value);
        } else if (typeof DEFAULT_VALUES[item.key] === 'number') {
          const n = parseInt(item.value, 10);
          current[item.key] = Number.isNaN(n) ? DEFAULT_VALUES[item.key] : n;
        } else {
          current[item.key] = item.value;
        }
      });
      setInputs(current);
      setInputsRow(current);
      if (refForm.current) {
        refForm.current.setValues(current);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadConfig();
  }, []);

  const onSubmit = async () => {
    const changes = compareObjects(inputs, inputsRow);
    // 脱敏占位（****... 开头）= 不修改已有 SecretKey，从提交队列中剔除。
    const effective = changes.filter((item) => {
      if (
        item.key === 'oss_image_setting.secret_key' &&
        typeof inputs[item.key] === 'string' &&
        inputs[item.key].startsWith(SECRET_MASK_PREFIX)
      ) {
        return false;
      }
      return true;
    });
    if (!effective.length) {
      showWarning(t('你似乎并没有修改什么'));
      return;
    }
    setSaving(true);
    const queue = effective.map((item) =>
      API.put('/api/option/', {
        key: item.key,
        value: String(inputs[item.key]),
      }),
    );
    try {
      const res = await Promise.all(queue);
      if (res.includes(undefined)) {
        showError(t('部分保存失败，请重试'));
        return;
      }
      showSuccess(t('保存成功'));
      await loadConfig();
    } catch (_err) {
      showError(t('保存失败，请重试'));
    } finally {
      setSaving(false);
    }
  };

  const buildProbePayload = () => {
    const stripPrefix = 'oss_image_setting.';
    const payload = {};
    for (const key of Object.keys(inputs)) {
      if (!key.startsWith(stripPrefix)) continue;
      const bare = key.slice(stripPrefix.length);
      payload[bare] = inputs[key];
    }
    // SecretKey 若仍为占位，发空串让后端沿用已保存值。
    if (
      typeof payload.secret_key === 'string' &&
      payload.secret_key.startsWith(SECRET_MASK_PREFIX)
    ) {
      payload.secret_key = '';
    }
    return payload;
  };

  const handlePing = async () => {
    setPinging(true);
    try {
      const res = await API.post('/api/oss/images/ping', buildProbePayload());
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

  const enabled = inputs['oss_image_setting.enabled'];

  return (
    <div style={{ padding: 16 }}>
      {!enabled && (
        <Banner
          type='info'
          description={t('当前功能未启用，开启开关后配置才会生效')}
          style={{ marginBottom: 16 }}
        />
      )}

      <Spin spinning={saving}>
        <Form
          values={inputs}
          getFormApi={(api) => (refForm.current = api)}
        >
          <Typography.Title heading={5}>{t('基础设置')}</Typography.Title>
          <Row gutter={16}>
            <Col xs={24} sm={12} md={8}>
              <Form.Switch
                field='oss_image_setting.enabled'
                label={t('启用图片 OSS 转存')}
                onChange={handleFieldChange('oss_image_setting.enabled')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.Switch
                field='oss_image_setting.fallback_to_upstream'
                label={t('失败时回退到上游 URL')}
                onChange={handleFieldChange(
                  'oss_image_setting.fallback_to_upstream',
                )}
              />
            </Col>
          </Row>

          <Typography.Title heading={5} style={{ marginTop: 20 }}>
            {t('MinIO 连接')}
          </Typography.Title>
          <Row gutter={16}>
            <Col xs={24} sm={12} md={8}>
              <Form.Input
                field='oss_image_setting.endpoint'
                label={t('Endpoint')}
                placeholder='127.0.0.1:9000'
                onChange={handleFieldChange('oss_image_setting.endpoint')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.Input
                field='oss_image_setting.access_key'
                label='AccessKey'
                onChange={handleFieldChange('oss_image_setting.access_key')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.Input
                field='oss_image_setting.secret_key'
                label='SecretKey'
                mode='password'
                placeholder={t('保持默认即沿用已保存值')}
                onChange={handleFieldChange('oss_image_setting.secret_key')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.Input
                field='oss_image_setting.bucket'
                label='Bucket'
                onChange={handleFieldChange('oss_image_setting.bucket')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.Input
                field='oss_image_setting.region'
                label='Region'
                onChange={handleFieldChange('oss_image_setting.region')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.Input
                field='oss_image_setting.public_url_prefix'
                label={t('公网 URL 前缀')}
                placeholder='https://cdn.example.com'
                onChange={handleFieldChange(
                  'oss_image_setting.public_url_prefix',
                )}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.Switch
                field='oss_image_setting.use_ssl'
                label='Use SSL'
                onChange={handleFieldChange('oss_image_setting.use_ssl')}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.Switch
                field='oss_image_setting.use_path_style'
                label='Path Style'
                onChange={handleFieldChange(
                  'oss_image_setting.use_path_style',
                )}
              />
            </Col>
          </Row>
          <Space style={{ marginTop: 12 }}>
            <Button onClick={handlePing} loading={pinging}>
              {t('连接测试')}
            </Button>
            <Text type='tertiary'>
              {t('连接测试使用当前表单值，不影响已保存配置')}
            </Text>
          </Space>

          <Typography.Title heading={5} style={{ marginTop: 20 }}>
            {t('生命周期')}
          </Typography.Title>
          <Row gutter={16}>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='oss_image_setting.retention_hours'
                label={t('保留小时数')}
                min={1}
                onChange={handleFieldChange(
                  'oss_image_setting.retention_hours',
                )}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='oss_image_setting.download_timeout_seconds'
                label={t('下载超时(秒)')}
                min={1}
                onChange={handleFieldChange(
                  'oss_image_setting.download_timeout_seconds',
                )}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='oss_image_setting.cleanup_interval_hours'
                label={t('清理周期(小时)')}
                min={1}
                extraText={t(
                  '修改后下次进程启动生效；如需立即生效请使用下方手动清理按钮',
                )}
                onChange={handleFieldChange(
                  'oss_image_setting.cleanup_interval_hours',
                )}
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='oss_image_setting.cleanup_batch_size'
                label={t('清理批大小')}
                min={1}
                onChange={handleFieldChange(
                  'oss_image_setting.cleanup_batch_size',
                )}
              />
            </Col>
          </Row>
          <Space style={{ marginTop: 12 }}>
            <Button onClick={handleCleanup} loading={cleaning}>
              {t('立即执行清理')}
            </Button>
          </Space>

          <Row style={{ marginTop: 20 }}>
            <Button theme='solid' type='primary' onClick={onSubmit}>
              {t('保存配置')}
            </Button>
          </Row>
        </Form>
      </Spin>
    </div>
  );
};

export default ImageOssSetting;
