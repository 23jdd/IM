import { AuthService, ChatService } from '../../bindings/im-client'
import { Events } from '@wailsio/runtime'

// 后端 TCP 地址（网关 :8000 或直连 :9000）。
const TCP_ADDR = '127.0.0.1:9000'

export const api = {
  // ---- HTTP (AuthService) ----
  login: (uid, password) => AuthService.Login(uid, password),
  register: (name, password, email, phone) =>
    AuthService.Register(name, password, email, phone),
  getProfile: (token) => AuthService.GetProfile(token),
  updateProfile: (token, profile) => AuthService.UpdateProfile(token, profile),
  changePassword: (token, oldPwd, newPwd) =>
    AuthService.ChangePassword(token, oldPwd, newPwd),
  uploadAvatar: (token, dataBase64, contentType) =>
    AuthService.UploadAvatar(token, dataBase64, contentType),
  getAvatar: (token, id) => AuthService.GetAvatar(token, id),
  getAvatarByUid: (token, uid) => AuthService.GetAvatarByUid(token, uid),
  getFriends: (token) => AuthService.GetFriends(token),
  getConversations: (token) => AuthService.GetConversations(token),
  groupCreate: (token, name, description) =>
    AuthService.GroupCreate(token, name, description),
  groupList: (token) => AuthService.GroupList(token),
  groupJoin: (token, groupId) => AuthService.GroupJoin(token, groupId),
  groupMembers: (token, groupId) => AuthService.GroupMembers(token, groupId),

  // ---- TCP 实时 (ChatService) ----
  connect: () => ChatService.Connect(TCP_ADDR),
  authTcp: (token) => ChatService.Auth(token),
  sendText: (toUid, content) => ChatService.SendText(toUid, content),
  sendGroupText: (groupId, content) => ChatService.SendGroupText(groupId, content),
  sync: () => ChatService.Sync(),
  disconnect: () => ChatService.Disconnect(),
}

// 监听后端推送事件，返回取消订阅函数。
export function onEvent(name, cb) {
  return Events.On(name, (e) => cb(e.data))
}

export const EVT = {
  STATUS: 'im:status',
  TEXT: 'im:text',
  OFFLINE: 'im:offline',
  ACK: 'im:ack',
  NACK: 'im:nack',
}
