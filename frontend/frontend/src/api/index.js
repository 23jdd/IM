import { AuthService, ChatService, LocalStore } from '../../bindings/im-client'
import { Events } from '@wailsio/runtime'

// 后端 TCP 地址（网关 :8000 或直连 :9000）。
const TCP_ADDR = '127.0.0.1:9000'

export const api = {
  // ---- HTTP (AuthService) ----
  login: (uid, password) => AuthService.Login(uid, password),
  register: (name, password, email, phone) =>
    AuthService.Register(name, password, email, phone),
  getProfile: (token) => AuthService.GetProfile(token),
  userInfo: (token, uid) => AuthService.UserInfo(token, uid),
  messageRecall: (token, msgId) => AuthService.MessageRecall(token, msgId),
  messageHistory: (token, peer, group, before, limit) =>
    AuthService.GetChatHistory(token, peer, group, before, limit),
  updateProfile: (token, profile) => AuthService.UpdateProfile(token, profile),
  changePassword: (token, oldPwd, newPwd) =>
    AuthService.ChangePassword(token, oldPwd, newPwd),
  uploadAvatar: (token, dataBase64, contentType) =>
    AuthService.UploadAvatar(token, dataBase64, contentType),
  getAvatar: (token, id) => AuthService.GetAvatar(token, id),
  getAvatarByUid: (token, uid) => AuthService.GetAvatarByUid(token, uid),
  uploadFile: (token, dataBase64, contentType) =>
    AuthService.UploadFile(token, dataBase64, contentType),
  saveFile: (suggestedName, dataBase64) =>
    ChatService.SaveFile(suggestedName, dataBase64),
  momentPublish: (token, content, images) =>
    AuthService.MomentPublish(token, content, images),
  momentTimeline: (token) => AuthService.MomentTimeline(token),
  momentLike: (token, momentId) => AuthService.MomentLike(token, momentId),
  momentComment: (token, momentId, content) =>
    AuthService.MomentComment(token, momentId, content),
  momentDelete: (token, momentId) => AuthService.MomentDelete(token, momentId),
  getFriends: (token) => AuthService.GetFriends(token),
  getConversations: (token) => AuthService.GetConversations(token),
  friendRequest: (token, friendUid, remark) =>
    AuthService.FriendRequest(token, friendUid, remark),
  friendRequests: (token) => AuthService.FriendRequests(token),
  friendAccept: (token, friendUid) => AuthService.FriendAccept(token, friendUid),
  friendRemove: (token, friendUid) => AuthService.FriendRemove(token, friendUid),
  friendBlock: (token, friendUid) => AuthService.FriendBlock(token, friendUid),
  friendUnblock: (token, friendUid) =>
    AuthService.FriendUnblock(token, friendUid),
  friendBlockList: (token) => AuthService.FriendBlockList(token),
  friendRemark: (token, friendUid, remark) =>
    AuthService.FriendRemark(token, friendUid, remark),
  groupCreate: (token, name, description) =>
    AuthService.GroupCreate(token, name, description),
  groupList: (token) => AuthService.GroupList(token),
  groupJoin: (token, groupId) => AuthService.GroupJoin(token, groupId),
  groupMembers: (token, groupId) => AuthService.GroupMembers(token, groupId),
  groupInvite: (token, groupId, friendUid) =>
    AuthService.GroupInvite(token, groupId, friendUid),
  groupJoinRequests: (token, groupId) =>
    AuthService.GroupJoinRequests(token, groupId),
  groupApprove: (token, groupId, applicantUid) =>
    AuthService.GroupApprove(token, groupId, applicantUid),
  groupReject: (token, groupId, applicantUid) =>
    AuthService.GroupReject(token, groupId, applicantUid),
  groupInfo: (token, groupId) => AuthService.GroupInfo(token, groupId),
  groupLeave: (token, groupId) => AuthService.GroupLeave(token, groupId),
  groupDisband: (token, groupId) => AuthService.GroupDisband(token, groupId),
  groupKick: (token, groupId, targetUid) =>
    AuthService.GroupKick(token, groupId, targetUid),
  groupTransfer: (token, groupId, targetUid) =>
    AuthService.GroupTransfer(token, groupId, targetUid),
  groupMute: (token, groupId, targetUid, minutes) =>
    AuthService.GroupMute(token, groupId, targetUid, minutes),
  groupAnnounce: (token, groupId, announcement) =>
    AuthService.GroupAnnounce(token, groupId, announcement),

  // ---- TCP 实时 (ChatService) ----
  connect: () => ChatService.Connect(TCP_ADDR),
  authTcp: (token) => ChatService.Auth(token),
  sendText: (toUid, content) => ChatService.SendText(toUid, content),
  sendGroupText: (groupId, content, mentions) =>
    ChatService.SendGroupText(groupId, content, mentions || []),
  sendTyping: (toUid, groupId) => ChatService.SendTyping(toUid, groupId),
  sendRead: (toUid, groupId, upTo) =>
    ChatService.SendRead(toUid, groupId, upTo),
  sync: () => ChatService.Sync(),
  disconnect: () => ChatService.Disconnect(),

  // ---- 本地 SQLite (LocalStore) ----
  localInit: (uid) => LocalStore.Init(uid),
  localSave: (peer, msgId, fromUid, content, self, status, ts) =>
    LocalStore.SaveMessage(peer, msgId, fromUid, content, self, status, ts),
  localLoad: (peer, limit) => LocalStore.LoadMessages(peer, limit),
  localRecall: (msgId) => LocalStore.MarkRecalled(msgId),
  saveSession: (token, uid, name, profile) =>
    LocalStore.SaveSession(token, uid, name, profile),
  loadSession: () => LocalStore.LoadSession(),
  clearSession: () => LocalStore.ClearSession(),
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
  NOTIFY: 'im:notify',
}
