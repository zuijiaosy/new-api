import React, { useEffect, useState, useRef } from 'react';
import {
  Button,
  Col,
  Form,
  Row,
  Spin,
  Table,
  Tag,
  Typography,
  Banner,
  Modal,
} from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

export default function SettingsQuotaReset(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [triggerLoading, setTriggerLoading] = useState(false);
  const [inputs, setInputs] = useState({
    QuotaResetEnabled: false,
    WeeklyQuotaLimitEnabled: false,
    QuotaResetTime: '00:01',
    QuotaResetConcurrency: 3,
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  // 保存配置
  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));

    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
        value = String(inputs[item.key]);
      } else {
        value = String(inputs[item.key]);
      }
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });

    setLoading(true);
    Promise.all(requestQueue)
      .then(() => {
        showSuccess(t('保存成功'));
        props.refresh();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  // 手动触发执行
  async function handleTrigger() {
    Modal.confirm({
      title: t('确认手动触发'),
      content: t('确定要立即执行额度重置任务吗？这将会重置所有活跃用户的额度。'),
      onOk: async () => {
        setTriggerLoading(true);
        try {
          const res = await API.post('/api/quota-reset/trigger');
          const { success, message } = res.data;
          if (success) {
            showSuccess(t('任务已触发，请稍后刷新查看执行结果'));
            // 延迟刷新日志
            setTimeout(() => {
              props.refresh();
            }, 2000);
          } else {
            showError(message || t('触发失败'));
          }
        } catch (error) {
          showError(t('触发失败'));
        } finally {
          setTriggerLoading(false);
        }
      },
    });
  }

  // 日志表格列定义
  const logColumns = [
    {
      title: t('执行时间'),
      dataIndex: 'executed_at',
      key: 'executed_at',
      render: (text) => {
        if (!text) return '-';
        const date = new Date(text);
        return date.toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' });
      },
    },
    {
      title: t('总用户数'),
      dataIndex: 'total_users',
      key: 'total_users',
    },
    {
      title: t('成功'),
      dataIndex: 'success_count',
      key: 'success_count',
      render: (text) => <Text type='success'>{text}</Text>,
    },
    {
      title: t('失败'),
      dataIndex: 'failed_count',
      key: 'failed_count',
      render: (text) =>
        text > 0 ? <Text type='danger'>{text}</Text> : <Text>{text}</Text>,
    },
    {
      title: t('跳过天卡'),
      dataIndex: 'skipped_day_card',
      key: 'skipped_day_card',
    },
    {
      title: t('耗时'),
      dataIndex: 'duration',
      key: 'duration',
    },
  ];

  useEffect(() => {
    const currentInputs = {};
    for (let key in inputs) {
      if (Object.keys(props.options).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    setInputs((prev) => ({ ...prev, ...currentInputs }));
    setInputsRow(structuredClone({ ...inputs, ...currentInputs }));
    if (refForm.current) {
      refForm.current.setValues({ ...inputs, ...currentInputs });
    }
  }, [props.options]);

  // 状态展示
  const renderStatus = () => {
    if (!props.status) return null;

    return (
      <Banner
        type={props.status.db_connected ? 'info' : 'warning'}
        description={
          <div>
            <Text>
              {t('数据库连接')}: {' '}
              {props.status.db_connected ? (
                <Tag color='green'>{t('已连接')}</Tag>
              ) : (
                <Tag color='red'>{t('未连接')}</Tag>
              )}
            </Text>
            <Text style={{ marginLeft: 16 }}>
              {t('功能状态')}: {' '}
              {props.status.enabled ? (
                <Tag color='green'>{t('已启用')}</Tag>
              ) : (
                <Tag color='grey'>{t('已禁用')}</Tag>
              )}
            </Text>
            <Text style={{ marginLeft: 16 }}>
              {t('周额度限制')}: {' '}
              {props.status.weekly_limit_enabled ? (
                <Tag color='green'>{t('已启用')}</Tag>
              ) : (
                <Tag color='grey'>{t('已禁用')}</Tag>
              )}
            </Text>
            <Text style={{ marginLeft: 16 }}>
              {t('执行状态')}: {' '}
              {props.status.is_running ? (
                <Tag color='orange'>{t('执行中')}</Tag>
              ) : (
                <Tag color='blue'>{t('空闲')}</Tag>
              )}
            </Text>
            <Text style={{ marginLeft: 16 }}>
              {t('重置时间')}: {props.status.reset_time || '00:01'}
            </Text>
          </div>
        }
        style={{ marginBottom: 16 }}
      />
    );
  };

  return (
    <Spin spinning={loading}>
      {renderStatus()}

      <Form
        values={inputs}
        getFormApi={(formAPI) => (refForm.current = formAPI)}
      >
        <Form.Section text={t('额度重置设置')}>
          <Row gutter={16}>
            <Col xs={24} sm={12} md={8}>
              <Form.Switch
                field='QuotaResetEnabled'
                label={t('启用每日额度重置')}
                extraText={t(
                  '每天定时重置用户的 API 额度，需要配置 CODEXZH_SQL_DSN 环境变量'
                )}
                checkedText='｜'
                uncheckedText='〇'
                onChange={(value) =>
                  setInputs({ ...inputs, QuotaResetEnabled: value })
                }
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.Switch
                field='WeeklyQuotaLimitEnabled'
                label={t('启用周额度限制')}
                extraText={t(
                  '启用后将根据用户的周额度限制计算每日可分配额度，关闭则仅使用日额度'
                )}
                checkedText='｜'
                uncheckedText='〇'
                onChange={(value) =>
                  setInputs({ ...inputs, WeeklyQuotaLimitEnabled: value })
                }
              />
            </Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} sm={12} md={8}>
              <Form.Input
                field='QuotaResetTime'
                label={t('重置执行时间')}
                placeholder='00:01'
                extraText={t('格式：HH:MM（北京时间），例如 00:01 表示每天凌晨 0 点 1 分执行')}
                onChange={(value) =>
                  setInputs({ ...inputs, QuotaResetTime: value })
                }
              />
            </Col>
            <Col xs={24} sm={12} md={8}>
              <Form.InputNumber
                field='QuotaResetConcurrency'
                label={t('并发处理数')}
                min={1}
                max={10}
                extraText={t('同时处理的用户数量，建议 3-5')}
                onChange={(value) =>
                  setInputs({ ...inputs, QuotaResetConcurrency: value })
                }
              />
            </Col>
          </Row>
          <Row style={{ marginTop: 16 }}>
            <Button
              size='default'
              type='primary'
              onClick={onSubmit}
              style={{ marginRight: 8 }}
            >
              {t('保存配置')}
            </Button>
            <Button
              size='default'
              type='secondary'
              onClick={handleTrigger}
              loading={triggerLoading}
              disabled={!props.status?.db_connected || props.status?.is_running}
            >
              {t('手动触发执行')}
            </Button>
            <Button
              size='default'
              type='tertiary'
              onClick={props.refresh}
              style={{ marginLeft: 8 }}
            >
              {t('刷新')}
            </Button>
          </Row>
        </Form.Section>

        <Form.Section text={t('执行日志')}>
          <Table
            columns={logColumns}
            dataSource={props.logs || []}
            rowKey='executed_at'
            pagination={false}
            size='small'
            empty={t('暂无执行记录')}
            expandedRowRender={(record) => {
              if (
                !record.error_messages ||
                record.error_messages.length === 0
              ) {
                return null;
              }
              return (
                <div style={{ padding: 8 }}>
                  <Text type='danger' strong>
                    {t('错误信息')}:
                  </Text>
                  <ul style={{ margin: '8px 0', paddingLeft: 20 }}>
                    {record.error_messages.slice(0, 10).map((msg, index) => (
                      <li key={index}>
                        <Text type='danger'>{msg}</Text>
                      </li>
                    ))}
                    {record.error_messages.length > 10 && (
                      <li>
                        <Text type='tertiary'>
                          {t('...还有 {{count}} 条错误', {
                            count: record.error_messages.length - 10,
                          })}
                        </Text>
                      </li>
                    )}
                  </ul>
                </div>
              );
            }}
            rowExpandable={(record) =>
              record.error_messages && record.error_messages.length > 0
            }
          />
        </Form.Section>
      </Form>
    </Spin>
  );
}
