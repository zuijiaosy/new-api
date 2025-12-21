import React, { useEffect, useState } from 'react';
import { Card, Spin } from '@douyinfe/semi-ui';
import { API, showError, toBoolean } from '../../helpers';
import SettingsQuotaReset from '../../pages/Setting/QuotaReset/SettingsQuotaReset';

const QuotaResetSetting = () => {
  const [inputs, setInputs] = useState({
    QuotaResetEnabled: false,
    WeeklyQuotaLimitEnabled: false,
    QuotaResetTime: '00:01',
    QuotaResetConcurrency: 3,
  });
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState(null);
  const [logs, setLogs] = useState([]);

  // 获取配置选项
  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        if (item.key in inputs) {
          if (item.key === 'QuotaResetEnabled' || item.key === 'WeeklyQuotaLimitEnabled') {
            newInputs[item.key] = toBoolean(item.value);
          } else if (item.key === 'QuotaResetConcurrency') {
            newInputs[item.key] = parseInt(item.value) || 3;
          } else {
            newInputs[item.key] = item.value;
          }
        }
      });
      setInputs((prev) => ({ ...prev, ...newInputs }));
    } else {
      showError(message);
    }
  };

  // 获取执行状态
  const getStatus = async () => {
    try {
      const res = await API.get('/api/quota-reset/status');
      const { success, data } = res.data;
      if (success) {
        setStatus(data);
      }
    } catch (error) {
      console.error('获取状态失败:', error);
    }
  };

  // 获取执行日志
  const getLogs = async () => {
    try {
      const res = await API.get('/api/quota-reset/logs?limit=20');
      const { success, data } = res.data;
      if (success) {
        setLogs(data || []);
      }
    } catch (error) {
      console.error('获取日志失败:', error);
    }
  };

  // 刷新所有数据
  const onRefresh = async () => {
    try {
      setLoading(true);
      await Promise.all([getOptions(), getStatus(), getLogs()]);
    } catch (error) {
      showError('刷新失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    onRefresh();
  }, []);

  return (
    <>
      <Spin spinning={loading} size='large'>
        <Card style={{ marginTop: '10px' }}>
          <SettingsQuotaReset
            options={inputs}
            status={status}
            logs={logs}
            refresh={onRefresh}
          />
        </Card>
      </Spin>
    </>
  );
};

export default QuotaResetSetting;
