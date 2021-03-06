package main

import (
	"net/http"
	"socialapi/models"
	"socialapi/request"
	"socialapi/rest"
	"socialapi/workers/common/tests"
	"testing"

	"github.com/koding/runner"
	. "github.com/smartystreets/goconvey/convey"
)

func TestChannelMessage(t *testing.T) {
	tests.WithRunner(t, func(r *runner.Runner) {
		Convey("While testing channel messages given a channel", t, func() {

			account, groupChannel, groupName := models.CreateRandomGroupDataWithChecks()

			nonOwnerAccount := models.CreateAccountInBothDbsWithCheck()

			nonOwnerSes, err := models.FetchOrCreateSession(nonOwnerAccount.Nick, groupName)
			So(err, ShouldBeNil)

			ses, err := models.FetchOrCreateSession(account.Nick, groupName)
			So(err, ShouldBeNil)

			Convey("message should be able added to the group channel", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)
				So(post.Id, ShouldNotEqual, 0)
				So(post.Body, ShouldNotEqual, "")
				Convey("message can be edited by owner", func() {

					initialPostBody := post.Body
					post.Body = "edited message"

					editedPost, err := rest.UpdatePost(post, ses.ClientId)
					So(err, ShouldBeNil)
					So(editedPost, ShouldNotBeNil)
					// body should not be same
					So(initialPostBody, ShouldNotEqual, editedPost.Body)
				})

				// for now social worker handles this issue
				Convey("message can be edited by an admin", nil)
				Convey("message can not be edited by non-owner", nil)

			})

			Convey("topic messages initialChannelId must be set as owner group channel id", func() {
				ses, err := models.FetchOrCreateSession(account.Nick, groupName)
				So(err, ShouldBeNil)
				So(ses, ShouldNotBeNil)

				topicChannel, err := rest.CreateChannelByGroupNameAndType(account.Id, "koding", models.Channel_TYPE_TOPIC, ses.ClientId)
				So(err, ShouldBeNil)
				So(topicChannel, ShouldNotBeNil)

				post, err := rest.CreatePost(topicChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)
				So(post.InitialChannelId, ShouldNotEqual, topicChannel.Id)
				publicChannel, err := rest.GetChannelWithToken(post.InitialChannelId, ses.ClientId)
				So(err, ShouldBeNil)
				So(publicChannel.TypeConstant, ShouldEqual, models.Channel_TYPE_GROUP)
			})

			Convey("message can be deleted by owner", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)
				err = rest.DeletePost(post.Id, ses.ClientId)
				So(err, ShouldBeNil)
				post2, err := rest.GetPost(post.Id, ses.ClientId)
				So(err, ShouldNotBeNil)
				So(post2, ShouldBeNil)
			})

			Convey("message should not have payload, if user does not allow", func() {
				h := http.Header{}
				h.Add("X-Forwarded-For", "208.72.139.54")
				post, err := rest.CreatePostWithHeader(groupChannel.Id, h, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)
				So(post.Payload, ShouldBeNil)
			})

			Convey("Message should have location if user allowed", func() {
				payload := make(map[string]interface{})
				payload["saveLocation"] = "Manisa"
				post, err := rest.CreatePostWithPayload(groupChannel.Id, payload, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)
				So(post.Payload, ShouldNotBeNil)
				So(*(post.Payload["saveLocation"]), ShouldEqual, "Manisa")
			})

			// handled by social worker
			Convey("message can be deleted by an admin", nil)
			Convey("message can not be edited by non-owner", nil)

			Convey("owner can like message", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)

				_, err = rest.AddInteraction("like", post.Id, post.AccountId, ses.ClientId)
				So(err, ShouldBeNil)

				cmc, err := rest.GetPostWithRelatedData(
					post.Id,
					&request.Query{
						AccountId: post.AccountId,
						GroupName: groupName,
					},
					ses.ClientId,
				)

				So(err, ShouldBeNil)
				So(cmc, ShouldNotBeNil)

				// it is liked by author
				So(cmc.Interactions["like"].IsInteracted, ShouldBeTrue)

				// actor length should be 1
				So(cmc.Interactions["like"].ActorsCount, ShouldEqual, 1)

			})

			Convey("non-owner can like message", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)

				_, err = rest.AddInteraction("like", post.Id, nonOwnerAccount.Id, nonOwnerSes.ClientId)
				So(err, ShouldBeNil)

				cmc, err := rest.GetPostWithRelatedData(
					post.Id,
					&request.Query{
						AccountId: nonOwnerAccount.Id,
						GroupName: groupName,
					},
					nonOwnerSes.ClientId,
				)

				So(err, ShouldBeNil)
				So(cmc, ShouldNotBeNil)

				// it is liked by author
				So(cmc.Interactions["like"].IsInteracted, ShouldBeTrue)

				// actor length should be 1
				So(cmc.Interactions["like"].ActorsCount, ShouldEqual, 1)

			})

			Convey("we should be able to get only interactions", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)

				_, err = rest.AddInteraction("like", post.Id, nonOwnerAccount.Id, nonOwnerSes.ClientId)
				So(err, ShouldBeNil)

				likes, err := rest.GetInteractions("like", post.Id)
				So(err, ShouldBeNil)

				So(len(likes), ShouldEqual, 1)

			})

			Convey("users should be able to  un-like message", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)

				_, err = rest.AddInteraction("like", post.Id, post.AccountId, ses.ClientId)
				So(err, ShouldBeNil)

				cmc, err := rest.GetPostWithRelatedData(
					post.Id,
					&request.Query{
						AccountId: post.AccountId,
						GroupName: groupName,
					},
					ses.ClientId,
				)

				So(err, ShouldBeNil)
				So(cmc, ShouldNotBeNil)

				// it is liked by author
				So(cmc.Interactions["like"].IsInteracted, ShouldBeTrue)

				// actor length should be 1
				So(cmc.Interactions["like"].ActorsCount, ShouldEqual, 1)

				err = rest.DeleteInteraction("like", post.Id, account.Id, ses.ClientId)
				So(err, ShouldBeNil)

				cmc, err = rest.GetPostWithRelatedData(
					post.Id,
					&request.Query{
						AccountId: post.AccountId,
						GroupName: groupName,
					},
					ses.ClientId,
				)

				So(err, ShouldBeNil)
				So(cmc, ShouldNotBeNil)

				// it is liked by author
				So(cmc.Interactions["like"].IsInteracted, ShouldBeFalse)

				// actor length should be 1
				So(cmc.Interactions["like"].ActorsCount, ShouldEqual, 0)
			})

			Convey("owner can post reply to message", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)

				reply, err := rest.AddReply(post.Id, groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(reply, ShouldNotBeNil)

				So(reply.AccountId, ShouldEqual, post.AccountId)

				cmc, err := rest.GetPostWithRelatedData(
					post.Id,
					&request.Query{
						AccountId: post.AccountId,
						GroupName: groupName,
					},
					ses.ClientId,
				)

				So(err, ShouldBeNil)
				So(cmc, ShouldNotBeNil)

				So(len(cmc.Replies), ShouldEqual, 1)

				So(cmc.Replies[0].Message.AccountId, ShouldEqual, post.AccountId)

			})

			Convey("we should be able to get only replies", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)

				reply, err := rest.AddReply(post.Id, groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(reply, ShouldNotBeNil)

				reply, err = rest.AddReply(post.Id, groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(reply, ShouldNotBeNil)

				replies, err := rest.GetReplies(post.Id, post.AccountId, groupName)
				So(err, ShouldBeNil)
				So(len(replies), ShouldEqual, 2)

			})

			Convey("we should be able to get replies with \"from\" query param", nil)

			Convey("non-owner can post reply to message", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)

				reply, err := rest.AddReply(post.Id, groupChannel.Id, nonOwnerSes.ClientId)
				So(err, ShouldBeNil)
				So(reply, ShouldNotBeNil)

				So(reply.AccountId, ShouldEqual, nonOwnerAccount.Id)

				cmc, err := rest.GetPostWithRelatedData(
					post.Id,
					&request.Query{
						AccountId: post.AccountId,
						GroupName: groupName,
					},
					ses.ClientId,
				)

				So(err, ShouldBeNil)
				So(cmc, ShouldNotBeNil)

				So(len(cmc.Replies), ShouldEqual, 1)

				So(cmc.Replies[0].Message.AccountId, ShouldEqual, nonOwnerAccount.Id)
			})

			Convey("reply can be liked", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)

				reply, err := rest.AddReply(post.Id, groupChannel.Id, nonOwnerSes.ClientId)
				So(err, ShouldBeNil)
				So(reply, ShouldNotBeNil)

				So(reply.AccountId, ShouldEqual, nonOwnerAccount.Id)

				_, err = rest.AddInteraction("like", reply.Id, nonOwnerAccount.Id, nonOwnerSes.ClientId)
				So(err, ShouldBeNil)

				cmc, err := rest.GetPostWithRelatedData(
					post.Id,
					&request.Query{
						AccountId: account.Id,
						GroupName: groupName,
					},
					ses.ClientId,
				)

				So(err, ShouldBeNil)
				So(cmc, ShouldNotBeNil)

				// it is liked by reply author, not post owner
				So(cmc.Interactions["like"].IsInteracted, ShouldBeFalse)

				// we didnt like the post, we liked the reply
				So(cmc.Interactions["like"].ActorsCount, ShouldEqual, 0)

				So(len(cmc.Replies), ShouldEqual, 1)

				// we liked the reply
				So(cmc.Replies[0].Interactions["like"].ActorsCount, ShouldEqual, 1)

				So(cmc.Replies[0].Interactions["like"].IsInteracted, ShouldBeFalse)

			})

			// for now those will be handled by social worker
			Convey("reply can be deleted by admin", nil)
			Convey("reply can not be deleted by non owner", nil)

			Convey("reply can be deleted by owner", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)

				sesNonOwner, err := models.FetchOrCreateSession(nonOwnerAccount.Nick, groupName)
				So(err, ShouldBeNil)

				reply, err := rest.AddReply(post.Id, groupChannel.Id, nonOwnerSes.ClientId)
				So(err, ShouldBeNil)
				So(reply, ShouldNotBeNil)

				err = rest.DeletePost(reply.Id, sesNonOwner.ClientId)
				So(err, ShouldBeNil)

				cmc, err := rest.GetPostWithRelatedData(
					post.Id,
					&request.Query{
						AccountId: account.Id,
						GroupName: groupName,
					},
					ses.ClientId,
				)

				So(err, ShouldBeNil)
				So(cmc, ShouldNotBeNil)

				So(len(cmc.Replies), ShouldEqual, 0)

			})

			Convey("while deleting message, also replies should be deleted", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)

				reply1, err := rest.AddReply(post.Id, groupChannel.Id, nonOwnerSes.ClientId)
				So(err, ShouldBeNil)
				So(reply1, ShouldNotBeNil)

				reply2, err := rest.AddReply(post.Id, groupChannel.Id, nonOwnerSes.ClientId)
				So(err, ShouldBeNil)
				So(reply2, ShouldNotBeNil)

				err = rest.DeletePost(post.Id, ses.ClientId)
				So(err, ShouldBeNil)

				cmc, err := rest.GetPostWithRelatedData(
					reply1.Id,
					&request.Query{
						AccountId: account.Id,
						GroupName: groupName,
					},
					ses.ClientId,
				)
				So(err, ShouldNotBeNil)
				So(cmc, ShouldBeNil)

				cmc, err = rest.GetPostWithRelatedData(
					reply2.Id,
					&request.Query{
						AccountId: account.Id,
						GroupName: groupName,
					},
					ses.ClientId,
				)

				So(err, ShouldNotBeNil)
				So(cmc, ShouldBeNil)

			})

			Convey("while deleting message replies' likes should be deleted", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)

				reply1, err := rest.AddReply(post.Id, groupChannel.Id, nonOwnerSes.ClientId)
				So(err, ShouldBeNil)
				So(reply1, ShouldNotBeNil)

				reply2, err := rest.AddReply(post.Id, groupChannel.Id, nonOwnerSes.ClientId)
				So(err, ShouldBeNil)
				So(reply2, ShouldNotBeNil)

				_, err = rest.AddInteraction("like", reply1.Id, account.Id, ses.ClientId)
				So(err, ShouldBeNil)

				_, err = rest.AddInteraction("like", reply2.Id, account.Id, ses.ClientId)
				So(err, ShouldBeNil)

				err = rest.DeletePost(post.Id, ses.ClientId)
				So(err, ShouldBeNil)

				interactions, err := rest.GetInteractions("like", reply1.Id)
				So(err, ShouldBeNil)
				So(interactions, ShouldNotBeNil)

				interactions, err = rest.GetInteractions("like", reply2.Id)
				So(err, ShouldBeNil)
				So(interactions, ShouldNotBeNil)

			})

			Convey("while deleting message, message likes should be deleted", func() {
				post, err := rest.CreatePost(groupChannel.Id, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)

				_, err = rest.AddInteraction("like", post.Id, account.Id, ses.ClientId)
				So(err, ShouldBeNil)

				err = rest.DeletePost(post.Id, ses.ClientId)
				So(err, ShouldBeNil)

				interactions, err := rest.GetInteractions("like", post.Id)
				So(err, ShouldBeNil)
				So(interactions, ShouldNotBeNil)
			})
			Convey("while deleting messages, they should be removed from all channels", nil)

			Convey("message can contain payload", func() {
				payload := make(map[string]interface{})
				payload["key1"] = "value1"
				payload["key2"] = 2
				payload["key3"] = true
				payload["key4"] = 3.4

				post, err := rest.CreatePostWithPayload(groupChannel.Id, payload, ses.ClientId)
				So(err, ShouldBeNil)
				So(post, ShouldNotBeNil)

				So(post.Payload, ShouldNotBeNil)
				So(*(post.Payload["key1"]), ShouldEqual, "value1")
				So(*(post.Payload["key2"]), ShouldEqual, "2")
				So(*(post.Payload["key3"]), ShouldEqual, "true")
				So(*(post.Payload["key4"]), ShouldEqual, "3.4")
			})

			// TODO before enabling this topic feed must be added to wercker
			// Convey("message should be fetched from all public channels with given slug", func() {
			// 	post, err := rest.CreatePost(groupChannel.Id, account.Id)
			// 	So(err, ShouldBeNil)
			// 	So(post, ShouldNotBeNil)

			// 	cmc, err := rest.GetPostBySlug(post.Slug, account.Id)
			// 	So(err, ShouldBeNil)
			// 	So(cmc.Message.Slug, ShouldEqual, post.Slug)

			// })
		})
	})
}
