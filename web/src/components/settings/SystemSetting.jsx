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

import React, { useEffect, useState, useRef } from 'react';
import {
  Button,
  Form,
  Row,
  Col,
  Typography,
  Modal,
  Banner,
  TagInput,
  Spin,
  Card,
  Radio,
  Select,
} from '@douyinfe/semi-ui';
const { Text } = Typography;
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
  toBoolean,
} from '../../helpers';
import axios from 'axios';
import { useTranslation } from 'react-i18next';

const SystemSetting = () => {
  const { t } = useTranslation();
  let [inputs, setInputs] = useState({
    PasswordLoginEnabled: '',
    PasswordRegisterEnabled: '',
    EmailVerificationEnabled: '',
    GitHubOAuthEnabled: '',
    GitHubClientId: '',
    GitHubClientSecret: '',
    'oidc.enabled': '',
    'oidc.client_id': '',
    'oidc.client_secret': '',
    'oidc.well_known': '',
    'oidc.authorization_endpoint': '',
    'oidc.token_endpoint': '',
    'oidc.user_info_endpoint': '',
    Notice: '',
    SMTPServer: '',
    SMTPPort: '',
    SMTPAccount: '',
    SMTPFrom: '',
    SMTPToken: '',
    WorkerUrl: '',
    WorkerValidKey: '',
    WorkerAllowHttpImageRequestEnabled: '',
    Footer: '',
    WeChatAuthEnabled: '',
    WeChatServerAddress: '',
    WeChatServerToken: '',
    WeChatAccountQRCodeImageURL: '',
    TurnstileCheckEnabled: '',
    TurnstileSiteKey: '',
    TurnstileSecretKey: '',
    RegisterEnabled: '',
    'passkey.enabled': '',
    'passkey.rp_display_name': '',
    'passkey.rp_id': '',
    'passkey.origins': [],
    'passkey.allow_insecure_origin': '',
    'passkey.user_verification': 'preferred',
    'passkey.attachment_preference': '',
    EmailDomainRestrictionEnabled: '',
    EmailAliasRestrictionEnabled: '',
    SMTPSSLEnabled: '',
    EmailDomainWhitelist: [],
    TelegramOAuthEnabled: '',
    TelegramBotToken: '',
    TelegramBotName: '',
    LinuxDOOAuthEnabled: '',
    LinuxDOClientId: '',
    LinuxDOClientSecret: '',
    LinuxDOMinimumTrustLevel: '',
    ServerAddress: '',
    // SSRF防护配置
    'fetch_setting.enable_ssrf_protection': true,
    'fetch_setting.allow_private_ip': '',
    'fetch_setting.domain_filter_mode': false, // true 白名单，false 黑名单
    'fetch_setting.ip_filter_mode': false, // true 白名单，false 黑名单
    'fetch_setting.domain_list': [],
    'fetch_setting.ip_list': [],
    'fetch_setting.allowed_ports': [],
    'fetch_setting.apply_ip_filter_for_domain': false,
  });

  const [originInputs, setOriginInputs] = useState({});
  const [loading, setLoading] = useState(false);
  const [isLoaded, setIsLoaded] = useState(false);
  const formApiRef = useRef(null);
  const [emailDomainWhitelist, setEmailDomainWhitelist] = useState([]);
  const [showPasswordLoginConfirmModal, setShowPasswordLoginConfirmModal] =
    useState(false);
  const [linuxDOOAuthEnabled, setLinuxDOOAuthEnabled] = useState(false);
  const [emailToAdd, setEmailToAdd] = useState('');
  const [domainFilterMode, setDomainFilterMode] = useState(true);
  const [ipFilterMode, setIpFilterMode] = useState(true);
  const [domainList, setDomainList] = useState([]);
  const [ipList, setIpList] = useState([]);
  const [allowedPorts, setAllowedPorts] = useState([]);

  const getOptions = async () => {
    setLoading(true);
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        switch (item.key) {
          case 'TopupGroupRatio':
            item.value = JSON.stringify(JSON.parse(item.value), null, 2);
            break;
          case 'EmailDomainWhitelist':
            setEmailDomainWhitelist(item.value ? item.value.split(',') : []);
            break;
          case 'fetch_setting.allow_private_ip':
          case 'fetch_setting.enable_ssrf_protection':
          case 'fetch_setting.domain_filter_mode':
          case 'fetch_setting.ip_filter_mode':
          case 'fetch_setting.apply_ip_filter_for_domain':
            item.value = toBoolean(item.value);
            break;
          case 'fetch_setting.domain_list':
            try {
              const domains = item.value ? JSON.parse(item.value) : [];
              setDomainList(Array.isArray(domains) ? domains : []);
            } catch (e) {
              setDomainList([]);
            }
            break;
          case 'fetch_setting.ip_list':
            try {
              const ips = item.value ? JSON.parse(item.value) : [];
              setIpList(Array.isArray(ips) ? ips : []);
            } catch (e) {
              setIpList([]);
            }
            break;
          case 'fetch_setting.allowed_ports':
            try {
              const ports = item.value ? JSON.parse(item.value) : [];
              setAllowedPorts(Array.isArray(ports) ? ports : []);
            } catch (e) {
              setAllowedPorts(['80', '443', '8080', '8443']);
            }
            break;
          case 'PasswordLoginEnabled':
          case 'PasswordRegisterEnabled':
          case 'EmailVerificationEnabled':
          case 'GitHubOAuthEnabled':
          case 'WeChatAuthEnabled':
          case 'TelegramOAuthEnabled':
          case 'RegisterEnabled':
          case 'TurnstileCheckEnabled':
          case 'EmailDomainRestrictionEnabled':
          case 'EmailAliasRestrictionEnabled':
          case 'SMTPSSLEnabled':
          case 'LinuxDOOAuthEnabled':
          case 'oidc.enabled':
          case 'passkey.enabled':
          case 'passkey.allow_insecure_origin':
          case 'WorkerAllowHttpImageRequestEnabled':
            item.value = toBoolean(item.value);
            break;
          case 'passkey.origins':
            // origins是逗号分隔的字符串，直接使用
            item.value = item.value || '';
            break;
          case 'passkey.rp_display_name':
          case 'passkey.rp_id':
          case 'passkey.attachment_preference':
            // 确保字符串字段不为null/undefined
            item.value = item.value || '';
            break;
          case 'passkey.user_verification':
            // 确保有默认值
            item.value = item.value || 'preferred';
            break;
          case 'Price':
          case 'MinTopUp':
            item.value = parseFloat(item.value);
            break;
          default:
            break;
        }
        newInputs[item.key] = item.value;
      });
      setInputs(newInputs);
      setOriginInputs(newInputs);
      // 同步模式布尔到本地状态
      if (
        typeof newInputs['fetch_setting.domain_filter_mode'] !== 'undefined'
      ) {
        setDomainFilterMode(!!newInputs['fetch_setting.domain_filter_mode']);
      }
      if (typeof newInputs['fetch_setting.ip_filter_mode'] !== 'undefined') {
        setIpFilterMode(!!newInputs['fetch_setting.ip_filter_mode']);
      }
      if (formApiRef.current) {
        formApiRef.current.setValues(newInputs);
      }
      setIsLoaded(true);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  useEffect(() => {
    getOptions();
  }, []);

  const updateOptions = async (options) => {
    setLoading(true);
    try {
      // 分离 checkbox 类型的选项和其他选项
      const checkboxOptions = options.filter((opt) =>
        opt.key.toLowerCase().endsWith('enabled'),
      );
      const otherOptions = options.filter(
        (opt) => !opt.key.toLowerCase().endsWith('enabled'),
      );

      // 处理 checkbox 类型的选项
      for (const opt of checkboxOptions) {
        const res = await API.put('/api/option/', {
          key: opt.key,
          value: opt.value.toString(),
        });
        if (!res.data.success) {
          showError(res.data.message);
          return;
        }
      }

      // 处理其他选项
      if (otherOptions.length > 0) {
        const requestQueue = otherOptions.map((opt) =>
          API.put('/api/option/', {
            key: opt.key,
            value:
              typeof opt.value === 'boolean' ? opt.value.toString() : opt.value,
          }),
        );

        const results = await Promise.all(requestQueue);

        // 检查所有请求是否成功
        const errorResults = results.filter((res) => !res.data.success);
        errorResults.forEach((res) => {
          showError(res.data.message);
        });
      }

      showSuccess(t('更新成功'));
      // 更新本地状态
      const newInputs = { ...inputs };
      options.forEach((opt) => {
        newInputs[opt.key] = opt.value;
      });
      setInputs(newInputs);
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
  };

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitWorker = async () => {
    let WorkerUrl = removeTrailingSlash(inputs.WorkerUrl);
    const options = [
      { key: 'WorkerUrl', value: WorkerUrl },
      {
        key: 'WorkerAllowHttpImageRequestEnabled',
        value: inputs.WorkerAllowHttpImageRequestEnabled ? 'true' : 'false',
      },
    ];
    if (inputs.WorkerValidKey !== '' || WorkerUrl === '') {
      options.push({ key: 'WorkerValidKey', value: inputs.WorkerValidKey });
    }
    await updateOptions(options);
  };

  const submitServerAddress = async () => {
    let ServerAddress = removeTrailingSlash(inputs.ServerAddress);
    await updateOptions([{ key: 'ServerAddress', value: ServerAddress }]);
  };

  const submitSMTP = async () => {
    const options = [];

    if (originInputs['SMTPServer'] !== inputs.SMTPServer) {
      options.push({ key: 'SMTPServer', value: inputs.SMTPServer });
    }
    if (originInputs['SMTPAccount'] !== inputs.SMTPAccount) {
      options.push({ key: 'SMTPAccount', value: inputs.SMTPAccount });
    }
    if (originInputs['SMTPFrom'] !== inputs.SMTPFrom) {
      options.push({ key: 'SMTPFrom', value: inputs.SMTPFrom });
    }
    if (
      originInputs['SMTPPort'] !== inputs.SMTPPort &&
      inputs.SMTPPort !== ''
    ) {
      options.push({ key: 'SMTPPort', value: inputs.SMTPPort });
    }
    if (
      originInputs['SMTPToken'] !== inputs.SMTPToken &&
      inputs.SMTPToken !== ''
    ) {
      options.push({ key: 'SMTPToken', value: inputs.SMTPToken });
    }

    if (options.length > 0) {
      await updateOptions(options);
    }
  };

  const submitEmailDomainWhitelist = async () => {
    if (Array.isArray(emailDomainWhitelist)) {
      await updateOptions([
        {
          key: 'EmailDomainWhitelist',
          value: emailDomainWhitelist.join(','),
        },
      ]);
    } else {
      showError(t('邮箱域名白名单格式不正确'));
    }
  };

  const submitSSRF = async () => {
    const options = [];

    // 处理域名过滤模式与列表
    options.push({
      key: 'fetch_setting.domain_filter_mode',
      value: domainFilterMode,
    });
    if (Array.isArray(domainList)) {
      options.push({
        key: 'fetch_setting.domain_list',
        value: JSON.stringify(domainList),
      });
    }

    // 处理IP过滤模式与列表
    options.push({
      key: 'fetch_setting.ip_filter_mode',
      value: ipFilterMode,
    });
    if (Array.isArray(ipList)) {
      options.push({
        key: 'fetch_setting.ip_list',
        value: JSON.stringify(ipList),
      });
    }

    // 处理端口配置
    if (Array.isArray(allowedPorts)) {
      options.push({
        key: 'fetch_setting.allowed_ports',
        value: JSON.stringify(allowedPorts),
      });
    }

    if (options.length > 0) {
      await updateOptions(options);
    }
  };

  const handleAddEmail = () => {
    if (emailToAdd && emailToAdd.trim() !== '') {
      const domain = emailToAdd.trim();

      // 验证域名格式
      const domainRegex =
        /^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/;
      if (!domainRegex.test(domain)) {
        showError(t('邮箱域名格式不正确，请输入有效的域名，如 gmail.com'));
        return;
      }

      // 检查是否已存在
      if (emailDomainWhitelist.includes(domain)) {
        showError(t('该域名已存在于白名单中'));
        return;
      }

      setEmailDomainWhitelist([...emailDomainWhitelist, domain]);
      setEmailToAdd('');
      showSuccess(t('已添加到白名单'));
    }
  };

  const submitWeChat = async () => {
    const options = [];

    if (originInputs['WeChatServerAddress'] !== inputs.WeChatServerAddress) {
      options.push({
        key: 'WeChatServerAddress',
        value: removeTrailingSlash(inputs.WeChatServerAddress),
      });
    }
    if (
      originInputs['WeChatAccountQRCodeImageURL'] !==
      inputs.WeChatAccountQRCodeImageURL
    ) {
      options.push({
        key: 'WeChatAccountQRCodeImageURL',
        value: inputs.WeChatAccountQRCodeImageURL,
      });
    }
    if (
      originInputs['WeChatServerToken'] !== inputs.WeChatServerToken &&
      inputs.WeChatServerToken !== ''
    ) {
      options.push({
        key: 'WeChatServerToken',
        value: inputs.WeChatServerToken,
      });
    }

    if (options.length > 0) {
      await updateOptions(options);
    }
  };

  const submitGitHubOAuth = async () => {
    const options = [];

    if (originInputs['GitHubClientId'] !== inputs.GitHubClientId) {
      options.push({ key: 'GitHubClientId', value: inputs.GitHubClientId });
    }
    if (
      originInputs['GitHubClientSecret'] !== inputs.GitHubClientSecret &&
      inputs.GitHubClientSecret !== ''
    ) {
      options.push({
        key: 'GitHubClientSecret',
        value: inputs.GitHubClientSecret,
      });
    }

    if (options.length > 0) {
      await updateOptions(options);
    }
  };

  const submitOIDCSettings = async () => {
    if (inputs['oidc.well_known'] && inputs['oidc.well_known'] !== '') {
      if (
        !inputs['oidc.well_known'].startsWith('http://') &&
        !inputs['oidc.well_known'].startsWith('https://')
      ) {
        showError(t('Well-Known URL 必须以 http:// 或 https:// 开头'));
        return;
      }
      try {
        const res = await axios.create().get(inputs['oidc.well_known']);
        inputs['oidc.authorization_endpoint'] =
          res.data['authorization_endpoint'];
        inputs['oidc.token_endpoint'] = res.data['token_endpoint'];
        inputs['oidc.user_info_endpoint'] = res.data['userinfo_endpoint'];
        showSuccess(t('获取 OIDC 配置成功！'));
      } catch (err) {
        console.error(err);
        showError(
          t('获取 OIDC 配置失败，请检查网络状况和 Well-Known URL 是否正确'),
        );
        return;
      }
    }

    const options = [];

    if (originInputs['oidc.well_known'] !== inputs['oidc.well_known']) {
      options.push({
        key: 'oidc.well_known',
        value: inputs['oidc.well_known'],
      });
    }
    if (originInputs['oidc.client_id'] !== inputs['oidc.client_id']) {
      options.push({ key: 'oidc.client_id', value: inputs['oidc.client_id'] });
    }
    if (
      originInputs['oidc.client_secret'] !== inputs['oidc.client_secret'] &&
      inputs['oidc.client_secret'] !== ''
    ) {
      options.push({
        key: 'oidc.client_secret',
        value: inputs['oidc.client_secret'],
      });
    }
    if (
      originInputs['oidc.authorization_endpoint'] !==
      inputs['oidc.authorization_endpoint']
    ) {
      options.push({
        key: 'oidc.authorization_endpoint',
        value: inputs['oidc.authorization_endpoint'],
      });
    }
    if (originInputs['oidc.token_endpoint'] !== inputs['oidc.token_endpoint']) {
      options.push({
        key: 'oidc.token_endpoint',
        value: inputs['oidc.token_endpoint'],
      });
    }
    if (
      originInputs['oidc.user_info_endpoint'] !==
      inputs['oidc.user_info_endpoint']
    ) {
      options.push({
        key: 'oidc.user_info_endpoint',
        value: inputs['oidc.user_info_endpoint'],
      });
    }

    if (options.length > 0) {
      await updateOptions(options);
    }
  };

  const submitTelegramSettings = async () => {
    const options = [
      { key: 'TelegramBotToken', value: inputs.TelegramBotToken },
      { key: 'TelegramBotName', value: inputs.TelegramBotName },
    ];
    await updateOptions(options);
  };

  const submitTurnstile = async () => {
    const options = [];

    if (originInputs['TurnstileSiteKey'] !== inputs.TurnstileSiteKey) {
      options.push({ key: 'TurnstileSiteKey', value: inputs.TurnstileSiteKey });
    }
    if (
      originInputs['TurnstileSecretKey'] !== inputs.TurnstileSecretKey &&
      inputs.TurnstileSecretKey !== ''
    ) {
      options.push({
        key: 'TurnstileSecretKey',
        value: inputs.TurnstileSecretKey,
      });
    }

    if (options.length > 0) {
      await updateOptions(options);
    }
  };

  const submitLinuxDOOAuth = async () => {
    const options = [];

    if (originInputs['LinuxDOClientId'] !== inputs.LinuxDOClientId) {
      options.push({ key: 'LinuxDOClientId', value: inputs.LinuxDOClientId });
    }
    if (
      originInputs['LinuxDOClientSecret'] !== inputs.LinuxDOClientSecret &&
      inputs.LinuxDOClientSecret !== ''
    ) {
      options.push({
        key: 'LinuxDOClientSecret',
        value: inputs.LinuxDOClientSecret,
      });
    }
    if (
      originInputs['LinuxDOMinimumTrustLevel'] !==
      inputs.LinuxDOMinimumTrustLevel
    ) {
      options.push({
        key: 'LinuxDOMinimumTrustLevel',
        value: inputs.LinuxDOMinimumTrustLevel,
      });
    }

    if (options.length > 0) {
      await updateOptions(options);
    }
  };

  const submitPasskeySettings = async () => {
    // 使用formApi直接获取当前表单值
    const formValues = formApiRef.current?.getValues() || {};

    const options = [];

    options.push({
      key: 'passkey.rp_display_name',
      value:
        formValues['passkey.rp_display_name'] ||
        inputs['passkey.rp_display_name'] ||
        '',
    });
    options.push({
      key: 'passkey.rp_id',
      value: formValues['passkey.rp_id'] || inputs['passkey.rp_id'] || '',
    });
    options.push({
      key: 'passkey.user_verification',
      value:
        formValues['passkey.user_verification'] ||
        inputs['passkey.user_verification'] ||
        'preferred',
    });
    options.push({
      key: 'passkey.attachment_preference',
      value:
        formValues['passkey.attachment_preference'] ||
        inputs['passkey.attachment_preference'] ||
        '',
    });
    options.push({
      key: 'passkey.origins',
      value: formValues['passkey.origins'] || inputs['passkey.origins'] || '',
    });

    await updateOptions(options);
  };

  const handleCheckboxChange = async (optionKey, event) => {
    const value = event.target.checked;

    if (optionKey === 'PasswordLoginEnabled' && !value) {
      setShowPasswordLoginConfirmModal(true);
    } else {
      await updateOptions([{ key: optionKey, value }]);
    }
    if (optionKey === 'LinuxDOOAuthEnabled') {
      setLinuxDOOAuthEnabled(value);
    }
  };

  const handlePasswordLoginConfirm = async () => {
    await updateOptions([{ key: 'PasswordLoginEnabled', value: false }]);
    setShowPasswordLoginConfirmModal(false);
  };

  return (
    <div>
      {isLoaded ? (
        <Form
          initValues={inputs}
          onValueChange={handleFormChange}
          getFormApi={(api) => (formApiRef.current = api)}
        >
          {({ formState, values, formApi }) => (
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: '10px',
                marginTop: '10px',
              }}
            >
              <Card>
                <Form.Section text={t('通用设置')}>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={24} lg={24} xl={24}>
                      <Form.Input
                        field='ServerAddress'
                        label={t('服务器地址')}
                        placeholder='https://yourdomain.com'
                        extraText={t(
                          '该服务器地址将影响支付回调地址以及默认首页展示的地址，请确保正确配置',
                        )}
                      />
                    </Col>
                  </Row>
                  <Button onClick={submitServerAddress}>
                    {t('更新服务器地址')}
                  </Button>
                </Form.Section>
              </Card>

              <Card>
                <Form.Section text={t('代理设置')}>
                  <Text>
                    （支持{' '}
                    <a
                      href='https://github.com/Calcium-Ion/new-api-worker'
                      target='_blank'
                      rel='noreferrer'
                    >
                      new-api-worker
                    </a>
                    ）
                  </Text>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field='WorkerUrl'
                        label={t('Worker地址')}
                        placeholder='例如：https://workername.yourdomain.workers.dev'
                      />
                    </Col>
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field='WorkerValidKey'
                        label={t('Worker密钥')}
                        placeholder='敏感信息不会发送到前端显示'
                        type='password'
                      />
                    </Col>
                  </Row>
                  <Form.Checkbox
                    field='WorkerAllowHttpImageRequestEnabled'
                    noLabel
                  >
                    {t('允许 HTTP 协议图片请求（适用于自部署代理）')}
                  </Form.Checkbox>
                  <Button onClick={submitWorker}>{t('更新Worker设置')}</Button>
                </Form.Section>
              </Card>

              <Card>
                <Form.Section text={t('SSRF防护设置')}>
                  <Text extraText={t('SSRF防护详细说明')}>
                    {t('配置服务器端请求伪造(SSRF)防护，用于保护内网资源安全')}
                  </Text>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={24} lg={24} xl={24}>
                      <Form.Checkbox
                        field='fetch_setting.enable_ssrf_protection'
                        noLabel
                        extraText={t('SSRF防护开关详细说明')}
                        onChange={(e) =>
                          handleCheckboxChange(
                            'fetch_setting.enable_ssrf_protection',
                            e,
                          )
                        }
                      >
                        {t('启用SSRF防护（推荐开启以保护服务器安全）')}
                      </Form.Checkbox>
                    </Col>
                  </Row>

                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                    style={{ marginTop: 16 }}
                  >
                    <Col xs={24} sm={24} md={24} lg={24} xl={24}>
                      <Form.Checkbox
                        field='fetch_setting.allow_private_ip'
                        noLabel
                        extraText={t('私有IP访问详细说明')}
                        onChange={(e) =>
                          handleCheckboxChange(
                            'fetch_setting.allow_private_ip',
                            e,
                          )
                        }
                      >
                        {t(
                          '允许访问私有IP地址（127.0.0.1、192.168.x.x等内网地址）',
                        )}
                      </Form.Checkbox>
                    </Col>
                  </Row>

                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                    style={{ marginTop: 16 }}
                  >
                    <Col xs={24} sm={24} md={24} lg={24} xl={24}>
                      <Form.Checkbox
                        field='fetch_setting.apply_ip_filter_for_domain'
                        noLabel
                        extraText={t('域名IP过滤详细说明')}
                        onChange={(e) =>
                          handleCheckboxChange(
                            'fetch_setting.apply_ip_filter_for_domain',
                            e,
                          )
                        }
                        style={{ marginBottom: 8 }}
                      >
                        {t('对域名启用 IP 过滤（实验性）')}
                      </Form.Checkbox>
                      <Text strong>
                        {t(domainFilterMode ? '域名白名单' : '域名黑名单')}
                      </Text>
                      <Text
                        type='secondary'
                        style={{ display: 'block', marginBottom: 8 }}
                      >
                        {t(
                          '支持通配符格式，如：example.com, *.api.example.com',
                        )}
                      </Text>
                      <Radio.Group
                        type='button'
                        value={domainFilterMode ? 'whitelist' : 'blacklist'}
                        onChange={(val) => {
                          const selected =
                            val && val.target ? val.target.value : val;
                          const isWhitelist = selected === 'whitelist';
                          setDomainFilterMode(isWhitelist);
                          setInputs((prev) => ({
                            ...prev,
                            'fetch_setting.domain_filter_mode': isWhitelist,
                          }));
                        }}
                        style={{ marginBottom: 8 }}
                      >
                        <Radio value='whitelist'>{t('白名单')}</Radio>
                        <Radio value='blacklist'>{t('黑名单')}</Radio>
                      </Radio.Group>
                      <TagInput
                        value={domainList}
                        onChange={(value) => {
                          setDomainList(value);
                          // 触发Form的onChange事件
                          setInputs((prev) => ({
                            ...prev,
                            'fetch_setting.domain_list': value,
                          }));
                        }}
                        placeholder={t('输入域名后回车，如：example.com')}
                        style={{ width: '100%' }}
                      />
                    </Col>
                  </Row>

                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                    style={{ marginTop: 16 }}
                  >
                    <Col xs={24} sm={24} md={24} lg={24} xl={24}>
                      <Text strong>
                        {t(ipFilterMode ? 'IP白名单' : 'IP黑名单')}
                      </Text>
                      <Text
                        type='secondary'
                        style={{ display: 'block', marginBottom: 8 }}
                      >
                        {t('支持CIDR格式，如：8.8.8.8, 192.168.1.0/24')}
                      </Text>
                      <Radio.Group
                        type='button'
                        value={ipFilterMode ? 'whitelist' : 'blacklist'}
                        onChange={(val) => {
                          const selected =
                            val && val.target ? val.target.value : val;
                          const isWhitelist = selected === 'whitelist';
                          setIpFilterMode(isWhitelist);
                          setInputs((prev) => ({
                            ...prev,
                            'fetch_setting.ip_filter_mode': isWhitelist,
                          }));
                        }}
                        style={{ marginBottom: 8 }}
                      >
                        <Radio value='whitelist'>{t('白名单')}</Radio>
                        <Radio value='blacklist'>{t('黑名单')}</Radio>
                      </Radio.Group>
                      <TagInput
                        value={ipList}
                        onChange={(value) => {
                          setIpList(value);
                          // 触发Form的onChange事件
                          setInputs((prev) => ({
                            ...prev,
                            'fetch_setting.ip_list': value,
                          }));
                        }}
                        placeholder={t('输入IP地址后回车，如：8.8.8.8')}
                        style={{ width: '100%' }}
                      />
                    </Col>
                  </Row>

                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                    style={{ marginTop: 16 }}
                  >
                    <Col xs={24} sm={24} md={24} lg={24} xl={24}>
                      <Text strong>{t('允许的端口')}</Text>
                      <Text
                        type='secondary'
                        style={{ display: 'block', marginBottom: 8 }}
                      >
                        {t('支持单个端口和端口范围，如：80, 443, 8000-8999')}
                      </Text>
                      <TagInput
                        value={allowedPorts}
                        onChange={(value) => {
                          setAllowedPorts(value);
                          // 触发Form的onChange事件
                          setInputs((prev) => ({
                            ...prev,
                            'fetch_setting.allowed_ports': value,
                          }));
                        }}
                        placeholder={t('输入端口后回车，如：80 或 8000-8999')}
                        style={{ width: '100%' }}
                      />
                      <Text
                        type='secondary'
                        style={{ display: 'block', marginBottom: 8 }}
                      >
                        {t('端口配置详细说明')}
                      </Text>
                    </Col>
                  </Row>

                  <Button onClick={submitSSRF} style={{ marginTop: 16 }}>
                    {t('更新SSRF防护设置')}
                  </Button>
                </Form.Section>
              </Card>

              <Card>
                <Form.Section text={t('配置登录注册')}>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Checkbox
                        field='PasswordLoginEnabled'
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange('PasswordLoginEnabled', e)
                        }
                      >
                        {t('允许通过密码进行登录')}
                      </Form.Checkbox>
                      <Form.Checkbox
                        field='PasswordRegisterEnabled'
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange('PasswordRegisterEnabled', e)
                        }
                      >
                        {t('允许通过密码进行注册')}
                      </Form.Checkbox>
                      <Form.Checkbox
                        field='EmailVerificationEnabled'
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange('EmailVerificationEnabled', e)
                        }
                      >
                        {t('通过密码注册时需要进行邮箱验证')}
                      </Form.Checkbox>
                      <Form.Checkbox
                        field='RegisterEnabled'
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange('RegisterEnabled', e)
                        }
                      >
                        {t('允许新用户注册')}
                      </Form.Checkbox>
                      <Form.Checkbox
                        field='TurnstileCheckEnabled'
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange('TurnstileCheckEnabled', e)
                        }
                      >
                        {t('允许 Turnstile 用户校验')}
                      </Form.Checkbox>
                    </Col>
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Checkbox
                        field='GitHubOAuthEnabled'
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange('GitHubOAuthEnabled', e)
                        }
                      >
                        {t('允许通过 GitHub 账户登录 & 注册')}
                      </Form.Checkbox>
                      <Form.Checkbox
                        field='LinuxDOOAuthEnabled'
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange('LinuxDOOAuthEnabled', e)
                        }
                      >
                        {t('允许通过 Linux DO 账户登录 & 注册')}
                      </Form.Checkbox>
                      <Form.Checkbox
                        field='WeChatAuthEnabled'
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange('WeChatAuthEnabled', e)
                        }
                      >
                        {t('允许通过微信登录 & 注册')}
                      </Form.Checkbox>
                      <Form.Checkbox
                        field='TelegramOAuthEnabled'
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange('TelegramOAuthEnabled', e)
                        }
                      >
                        {t('允许通过 Telegram 进行登录')}
                      </Form.Checkbox>
                      <Form.Checkbox
                        field="['oidc.enabled']"
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange('oidc.enabled', e)
                        }
                      >
                        {t('允许通过 OIDC 进行登录')}
                      </Form.Checkbox>
                    </Col>
                  </Row>
                </Form.Section>
              </Card>

              <Card>
                <Form.Section text={t('配置 Passkey')}>
                  <Text>{t('用以支持基于 WebAuthn 的无密码登录注册')}</Text>
                  <Banner
                    type='info'
                    description={t(
                      'Passkey 是基于 WebAuthn 标准的无密码身份验证方法，支持指纹、面容、硬件密钥等认证方式',
                    )}
                    style={{ marginBottom: 20, marginTop: 16 }}
                  />
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={24} lg={24} xl={24}>
                      <Form.Checkbox
                        field="['passkey.enabled']"
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange('passkey.enabled', e)
                        }
                      >
                        {t('允许通过 Passkey 登录 & 认证')}
                      </Form.Checkbox>
                    </Col>
                  </Row>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field="['passkey.rp_display_name']"
                        label={t('服务显示名称')}
                        placeholder={t('默认使用系统名称')}
                        extraText={t(
                          "用户注册时看到的网站名称，比如'我的网站'",
                        )}
                      />
                    </Col>
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field="['passkey.rp_id']"
                        label={t('网站域名标识')}
                        placeholder={t('例如：example.com')}
                        extraText={t(
                          '留空则默认使用服务器地址，注意不能携带http://或者https://',
                        )}
                      />
                    </Col>
                  </Row>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                    style={{ marginTop: 16 }}
                  >
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Select
                        field="['passkey.user_verification']"
                        label={t('安全验证级别')}
                        placeholder={t('是否要求指纹/面容等生物识别')}
                        optionList={[
                          {
                            label: t('推荐使用（用户可选）'),
                            value: 'preferred',
                          },
                          { label: t('强制要求'), value: 'required' },
                          { label: t('不建议使用'), value: 'discouraged' },
                        ]}
                        extraText={t('推荐：用户可以选择是否使用指纹等验证')}
                      />
                    </Col>
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Select
                        field="['passkey.attachment_preference']"
                        label={t('设备类型偏好')}
                        placeholder={t('选择支持的认证设备类型')}
                        optionList={[
                          { label: t('不限制'), value: '' },
                          { label: t('本设备内置'), value: 'platform' },
                          { label: t('外接设备'), value: 'cross-platform' },
                        ]}
                        extraText={t(
                          '本设备：手机指纹/面容，外接：USB安全密钥',
                        )}
                      />
                    </Col>
                  </Row>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                    style={{ marginTop: 16 }}
                  >
                    <Col xs={24} sm={24} md={24} lg={24} xl={24}>
                      <Form.Checkbox
                        field="['passkey.allow_insecure_origin']"
                        noLabel
                        extraText={t('仅用于开发环境，生产环境应使用 HTTPS')}
                        onChange={(e) =>
                          handleCheckboxChange(
                            'passkey.allow_insecure_origin',
                            e,
                          )
                        }
                      >
                        {t('允许不安全的 Origin（HTTP）')}
                      </Form.Checkbox>
                    </Col>
                  </Row>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                    style={{ marginTop: 16 }}
                  >
                    <Col xs={24} sm={24} md={24} lg={24} xl={24}>
                      <Form.Input
                        field="['passkey.origins']"
                        label={t('允许的 Origins')}
                        placeholder={t('填写带https的域名，逗号分隔')}
                        extraText={t(
                          '为空则默认使用服务器地址，多个 Origin 用逗号分隔，例如 https://newapi.pro,https://newapi.com ,注意不能携带[]，需使用https',
                        )}
                      />
                    </Col>
                  </Row>
                  <Button
                    onClick={submitPasskeySettings}
                    style={{ marginTop: 16 }}
                  >
                    {t('保存 Passkey 设置')}
                  </Button>
                </Form.Section>
              </Card>

              <Card>
                <Form.Section text={t('配置邮箱域名白名单')}>
                  <Text>{t('用以防止恶意用户利用临时邮箱批量注册')}</Text>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Checkbox
                        field='EmailDomainRestrictionEnabled'
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange(
                            'EmailDomainRestrictionEnabled',
                            e,
                          )
                        }
                      >
                        启用邮箱域名白名单
                      </Form.Checkbox>
                    </Col>
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Checkbox
                        field='EmailAliasRestrictionEnabled'
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange(
                            'EmailAliasRestrictionEnabled',
                            e,
                          )
                        }
                      >
                        启用邮箱别名限制
                      </Form.Checkbox>
                    </Col>
                  </Row>
                  <TagInput
                    value={emailDomainWhitelist}
                    onChange={setEmailDomainWhitelist}
                    placeholder={t('输入域名后回车')}
                    style={{ width: '100%', marginTop: 16 }}
                  />
                  <Form.Input
                    placeholder={t('输入要添加的邮箱域名')}
                    value={emailToAdd}
                    onChange={(value) => setEmailToAdd(value)}
                    style={{ marginTop: 16 }}
                    suffix={
                      <Button
                        theme='solid'
                        type='primary'
                        onClick={handleAddEmail}
                      >
                        {t('添加')}
                      </Button>
                    }
                    onEnterPress={handleAddEmail}
                  />
                  <Button
                    onClick={submitEmailDomainWhitelist}
                    style={{ marginTop: 10 }}
                  >
                    {t('保存邮箱域名白名单设置')}
                  </Button>
                </Form.Section>
              </Card>
              <Card>
                <Form.Section text={t('配置 SMTP')}>
                  <Text>{t('用以支持系统的邮件发送')}</Text>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                      <Form.Input
                        field='SMTPServer'
                        label={t('SMTP 服务器地址')}
                      />
                    </Col>
                    <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                      <Form.Input field='SMTPPort' label={t('SMTP 端口')} />
                    </Col>
                    <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                      <Form.Input field='SMTPAccount' label={t('SMTP 账户')} />
                    </Col>
                  </Row>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                    style={{ marginTop: 16 }}
                  >
                    <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                      <Form.Input
                        field='SMTPFrom'
                        label={t('SMTP 发送者邮箱')}
                      />
                    </Col>
                    <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                      <Form.Input
                        field='SMTPToken'
                        label={t('SMTP 访问凭证')}
                        type='password'
                        placeholder='敏感信息不会发送到前端显示'
                      />
                    </Col>
                    <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                      <Form.Checkbox
                        field='SMTPSSLEnabled'
                        noLabel
                        onChange={(e) =>
                          handleCheckboxChange('SMTPSSLEnabled', e)
                        }
                      >
                        {t('启用SMTP SSL')}
                      </Form.Checkbox>
                    </Col>
                  </Row>
                  <Button onClick={submitSMTP}>{t('保存 SMTP 设置')}</Button>
                </Form.Section>
              </Card>
              <Card>
                <Form.Section text={t('配置 OIDC')}>
                  <Text>
                    {t(
                      '用以支持通过 OIDC 登录，例如 Okta、Auth0 等兼容 OIDC 协议的 IdP',
                    )}
                  </Text>
                  <Banner
                    type='info'
                    description={`${t('主页链接填')} ${inputs.ServerAddress ? inputs.ServerAddress : t('网站地址')}，${t('重定向 URL 填')} ${inputs.ServerAddress ? inputs.ServerAddress : t('网站地址')}/oauth/oidc`}
                    style={{ marginBottom: 20, marginTop: 16 }}
                  />
                  <Text>
                    {t(
                      '若你的 OIDC Provider 支持 Discovery Endpoint，你可以仅填写 OIDC Well-Known URL，系统会自动获取 OIDC 配置',
                    )}
                  </Text>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field="['oidc.well_known']"
                        label={t('Well-Known URL')}
                        placeholder={t('请输入 OIDC 的 Well-Known URL')}
                      />
                    </Col>
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field="['oidc.client_id']"
                        label={t('Client ID')}
                        placeholder={t('输入 OIDC 的 Client ID')}
                      />
                    </Col>
                  </Row>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field="['oidc.client_secret']"
                        label={t('Client Secret')}
                        type='password'
                        placeholder={t('敏感信息不会发送到前端显示')}
                      />
                    </Col>
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field="['oidc.authorization_endpoint']"
                        label={t('Authorization Endpoint')}
                        placeholder={t('输入 OIDC 的 Authorization Endpoint')}
                      />
                    </Col>
                  </Row>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field="['oidc.token_endpoint']"
                        label={t('Token Endpoint')}
                        placeholder={t('输入 OIDC 的 Token Endpoint')}
                      />
                    </Col>
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field="['oidc.user_info_endpoint']"
                        label={t('User Info Endpoint')}
                        placeholder={t('输入 OIDC 的 Userinfo Endpoint')}
                      />
                    </Col>
                  </Row>
                  <Button onClick={submitOIDCSettings}>
                    {t('保存 OIDC 设置')}
                  </Button>
                </Form.Section>
              </Card>

              <Card>
                <Form.Section text={t('配置 GitHub OAuth App')}>
                  <Text>{t('用以支持通过 GitHub 进行登录注册')}</Text>
                  <Banner
                    type='info'
                    description={`${t('Homepage URL 填')} ${inputs.ServerAddress ? inputs.ServerAddress : t('网站地址')}，${t('Authorization callback URL 填')} ${inputs.ServerAddress ? inputs.ServerAddress : t('网站地址')}/oauth/github`}
                    style={{ marginBottom: 20, marginTop: 16 }}
                  />
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field='GitHubClientId'
                        label={t('GitHub Client ID')}
                      />
                    </Col>
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field='GitHubClientSecret'
                        label={t('GitHub Client Secret')}
                        type='password'
                        placeholder={t('敏感信息不会发送到前端显示')}
                      />
                    </Col>
                  </Row>
                  <Button onClick={submitGitHubOAuth}>
                    {t('保存 GitHub OAuth 设置')}
                  </Button>
                </Form.Section>
              </Card>
              <Card>
                <Form.Section text={t('配置 Linux DO OAuth')}>
                  <Text>
                    {t('用以支持通过 Linux DO 进行登录注册')}
                    <a
                      href='https://connect.linux.do/'
                      target='_blank'
                      rel='noreferrer'
                      style={{
                        display: 'inline-block',
                        marginLeft: 4,
                        marginRight: 4,
                      }}
                    >
                      {t('点击此处')}
                    </a>
                    {t('管理你的 LinuxDO OAuth App')}
                  </Text>
                  <Banner
                    type='info'
                    description={`${t('回调 URL 填')} ${inputs.ServerAddress ? inputs.ServerAddress : t('网站地址')}/oauth/linuxdo`}
                    style={{ marginBottom: 20, marginTop: 16 }}
                  />
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={10} lg={10} xl={10}>
                      <Form.Input
                        field='LinuxDOClientId'
                        label={t('Linux DO Client ID')}
                        placeholder={t('输入你注册的 LinuxDO OAuth APP 的 ID')}
                      />
                    </Col>
                    <Col xs={24} sm={24} md={10} lg={10} xl={10}>
                      <Form.Input
                        field='LinuxDOClientSecret'
                        label={t('Linux DO Client Secret')}
                        type='password'
                        placeholder={t('敏感信息不会发送到前端显示')}
                      />
                    </Col>
                    <Col xs={24} sm={24} md={4} lg={4} xl={4}>
                      <Form.Input
                        field='LinuxDOMinimumTrustLevel'
                        label='LinuxDO Minimum Trust Level'
                        placeholder='允许注册的最低信任等级'
                      />
                    </Col>
                  </Row>
                  <Button onClick={submitLinuxDOOAuth}>
                    {t('保存 Linux DO OAuth 设置')}
                  </Button>
                </Form.Section>
              </Card>

              <Card>
                <Form.Section text={t('配置 WeChat Server')}>
                  <Text>{t('用以支持通过微信进行登录注册')}</Text>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                      <Form.Input
                        field='WeChatServerAddress'
                        label={t('WeChat Server 服务器地址')}
                      />
                    </Col>
                    <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                      <Form.Input
                        field='WeChatServerToken'
                        label={t('WeChat Server 访问凭证')}
                        type='password'
                        placeholder={t('敏感信息不会发送到前端显示')}
                      />
                    </Col>
                    <Col xs={24} sm={24} md={8} lg={8} xl={8}>
                      <Form.Input
                        field='WeChatAccountQRCodeImageURL'
                        label={t('微信公众号二维码图片链接')}
                      />
                    </Col>
                  </Row>
                  <Button onClick={submitWeChat}>
                    {t('保存 WeChat Server 设置')}
                  </Button>
                </Form.Section>
              </Card>

              <Card>
                <Form.Section text={t('配置 Telegram 登录')}>
                  <Text>{t('用以支持通过 Telegram 进行登录注册')}</Text>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field='TelegramBotToken'
                        label={t('Telegram Bot Token')}
                        placeholder={t('敏感信息不会发送到前端显示')}
                        type='password'
                      />
                    </Col>
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field='TelegramBotName'
                        label={t('Telegram Bot 名称')}
                      />
                    </Col>
                  </Row>
                  <Button onClick={submitTelegramSettings}>
                    {t('保存 Telegram 登录设置')}
                  </Button>
                </Form.Section>
              </Card>

              <Card>
                <Form.Section text={t('配置 Turnstile')}>
                  <Text>{t('用以支持用户校验')}</Text>
                  <Row
                    gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
                  >
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field='TurnstileSiteKey'
                        label={t('Turnstile Site Key')}
                      />
                    </Col>
                    <Col xs={24} sm={24} md={12} lg={12} xl={12}>
                      <Form.Input
                        field='TurnstileSecretKey'
                        label={t('Turnstile Secret Key')}
                        type='password'
                        placeholder={t('敏感信息不会发送到前端显示')}
                      />
                    </Col>
                  </Row>
                  <Button onClick={submitTurnstile}>
                    {t('保存 Turnstile 设置')}
                  </Button>
                </Form.Section>
              </Card>

              <Modal
                title={t('确认取消密码登录')}
                visible={showPasswordLoginConfirmModal}
                onOk={handlePasswordLoginConfirm}
                onCancel={() => {
                  setShowPasswordLoginConfirmModal(false);
                  formApiRef.current.setValue('PasswordLoginEnabled', true);
                }}
                okText={t('确认')}
                cancelText={t('取消')}
              >
                <p>
                  {t(
                    '您确定要取消密码登录功能吗？这可能会影响用户的登录方式。',
                  )}
                </p>
              </Modal>
            </div>
          )}
        </Form>
      ) : (
        <div
          style={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            height: '100vh',
          }}
        >
          <Spin size='large' />
        </div>
      )}
    </div>
  );
};

export default SystemSetting;
