// import React, { useState, useEffect } from "react";
// import CreateGroupModal from "../components/CreateGroupModal";
// import ApplyGroupModal from "../components/ApplyGroupModal";
// import UpdateGroupInfoModal from "../components/UpdateGroupInfoModal";
// import RemoveGroupMembersModal from "../components/RemoveGroupMembersModal";
// import { leaveGroup, dismissGroup } from "../hooks/useGroupActions";
// import axios from "axios";

// interface Group {
//   uuid: string;
//   name: string;
//   avatar: string;
// }

// const ChatPage: React.FC = () => {
//   const [myGroups, setMyGroups] = useState<Group[]>([]);
//   const [joinedGroups, setJoinedGroups] = useState<Group[]>([]);
//   const [currentGroup, setCurrentGroup] = useState<Group | null>(null);

//   const [showCreateModal, setShowCreateModal] = useState(false);
//   const [showApplyModal, setShowApplyModal] = useState(false);
//   const [showUpdateModal, setShowUpdateModal] = useState(false);
//   const [showRemoveModal, setShowRemoveModal] = useState(false);

//   const userUuid = "user-uuid-xxx"; // TODO: 从登录态获取

//   // 加载我创建的群聊
//   const loadMyGroups = async () => {
//     try {
//       const res = await axios.post("http://localhost:8080/group/loadMyGroup");
//       if (res.data.groups) {
//         setMyGroups(res.data.groups);
//       }
//     } catch (err) {
//       console.error("加载群聊失败", err);
//     }
//   };

//   useEffect(() => {
//     loadMyGroups();
//     // TODO: 加载“我加入的群聊”接口
//   }, []);

//   return (
//     <div className="flex h-screen">
//       {/* 左侧群聊列表 */}
//       <aside className="w-64 bg-gray-100 p-4 space-y-4 overflow-y-auto border-r">
//         <h3 className="text-lg font-bold">我创建的群聊</h3>
//         {myGroups.map((g) => (
//           <div
//             key={g.uuid}
//             className={`flex items-center gap-2 p-2 rounded cursor-pointer hover:bg-gray-200 ${
//               currentGroup?.uuid === g.uuid ? "bg-gray-300" : ""
//             }`}
//             onClick={() => setCurrentGroup(g)}
//           >
//             <img src={g.avatar} alt="" className="w-8 h-8 rounded-md" />
//             <span>{g.name}</span>
//           </div>
//         ))}

//         <h3 className="text-lg font-bold mt-6">我加入的群聊</h3>
//         {joinedGroups.map((g) => (
//           <div
//             key={g.uuid}
//             className={`flex items-center gap-2 p-2 rounded cursor-pointer hover:bg-gray-200 ${
//               currentGroup?.uuid === g.uuid ? "bg-gray-300" : ""
//             }`}
//             onClick={() => setCurrentGroup(g)}
//           >
//             <img src={g.avatar} alt="" className="w-8 h-8 rounded-md" />
//             <span>{g.name}</span>
//           </div>
//         ))}

//         <div className="mt-6 space-y-2">
//           <button
//             onClick={() => setShowCreateModal(true)}
//             className="w-full bg-blue-500 text-white py-2 rounded hover:bg-blue-600"
//           >
//             创建群聊
//           </button>
//           <button
//             onClick={() => setShowApplyModal(true)}
//             className="w-full bg-green-500 text-white py-2 rounded hover:bg-green-600"
//           >
//             申请加群
//           </button>
//         </div>
//       </aside>

//       {/* 中间聊天窗口 */}
//       <main className="flex-1 flex flex-col">
//         {currentGroup ? (
//           <>
//             <header className="flex justify-between items-center p-4 border-b">
//               <h2 className="text-xl font-bold">{currentGroup.name}</h2>
//               <div className="space-x-2">
//                 <button
//                   onClick={() => setShowUpdateModal(true)}
//                   className="bg-yellow-500 text-white px-3 py-1 rounded hover:bg-yellow-600"
//                 >
//                   修改资料
//                 </button>
//                 <button
//                   onClick={() => setShowRemoveModal(true)}
//                   className="bg-orange-500 text-white px-3 py-1 rounded hover:bg-orange-600"
//                 >
//                   移除成员
//                 </button>
//                 <button
//                   onClick={() => dismissGroup(currentGroup.uuid)}
//                   className="bg-red-500 text-white px-3 py-1 rounded hover:bg-red-600"
//                 >
//                   解散群聊
//                 </button>
//                 <button
//                   onClick={() => leaveGroup(currentGroup.uuid)}
//                   className="bg-gray-500 text-white px-3 py-1 rounded hover:bg-gray-600"
//                 >
//                   退出群聊
//                 </button>
//               </div>
//             </header>
//             <section className="flex-1 p-4">
//               <p className="text-gray-500">这里是聊天消息区域（TODO: WebSocket）</p>
//             </section>
//           </>
//         ) : (
//           <div className="flex items-center justify-center flex-1">
//             <p className="text-gray-400">请选择一个群聊开始聊天</p>
//           </div>
//         )}
//       </main>

//       {/* 模态框们 */}
//       <CreateGroupModal
//         isOpen={showCreateModal}
//         onClose={() => setShowCreateModal(false)}
//         ownerId={userUuid}
//         onSuccess={loadMyGroups}
//       />

//       <ApplyGroupModal
//         isOpen={showApplyModal}
//         onClose={() => setShowApplyModal(false)}
//         userId={userUuid}
//       />

//       {currentGroup && (
//         <UpdateGroupInfoModal
//           isOpen={showUpdateModal}
//           onClose={() => setShowUpdateModal(false)}
//           groupId={currentGroup.uuid}
//           onSuccess={loadMyGroups}
//         />
//       )}

//       {currentGroup && (
//         <RemoveGroupMembersModal
//           isOpen={showRemoveModal}
//           onClose={() => setShowRemoveModal(false)}
//           groupId={currentGroup.uuid}
//           onSuccess={loadMyGroups}
//         />
//       )}
//     </div>
//   );
// };

// export default ChatPage;
