/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState } from 'react';
import { Button, InputNumber, Space, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../helpers';

const { Text } = Typography;

const TokenRPMSettings = ({
  defaultTokenRPM,
  setDefaultTokenRPM,
  refreshTokenRPMOverview,
  t,
}) => {
  const [loading, setLoading] = useState(false);
  const [localRPM, setLocalRPM] = useState(defaultTokenRPM || 0);

  useEffect(() => {
    setLocalRPM(defaultTokenRPM || 0);
  }, [defaultTokenRPM]);

  const onSave = async () => {
    const rpm = Number(localRPM || 0);
    if (!Number.isInteger(rpm) || rpm < 0) {
      showError(t('默认 RPM 必须是大于等于 0 的整数'));
      return;
    }
    setLoading(true);
    try {
      const res =
        rpm > 0
          ? await API.put('/api/token/rpm/default', { rpm })
          : await API.delete('/api/token/rpm/default');
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setDefaultTokenRPM(rpm);
      await refreshTokenRPMOverview();
      showSuccess(t('默认 RPM 保存成功'));
    } catch (error) {
      showError(error.message || t('默认 RPM 保存失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className='flex flex-wrap items-center gap-2'>
      <Text strong>{t('默认 RPM')}</Text>
      <InputNumber
        min={0}
        step={1}
        value={localRPM}
        onChange={(value) => setLocalRPM(value || 0)}
        style={{ width: 140 }}
        placeholder={t('0 表示不限制')}
      />
      <Button size='small' loading={loading} onClick={onSave}>
        {t('保存默认 RPM')}
      </Button>
      <Space>
        <Text type='secondary'>
          {t('未单独设置的令牌将继承这里的值，0 表示不限制')}
        </Text>
      </Space>
    </div>
  );
};

export default TokenRPMSettings;
